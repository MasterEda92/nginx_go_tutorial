# Go-App hinter nginx (Reverse-Proxy, HTTPS, Firewall)

Protokoll aller Schritte (Ubuntu 26.10, arm64, nginx 1.30, Go 1.27, ufw).

**Zielbild:**

```
Browser ──HTTPS:443──▶ nginx ──HTTP:8081 (nur localhost)──▶ Go-App
              ▲
        ufw erlaubt 443
```

Warum ein Reverse-Proxy? nginx übernimmt TLS-Verschlüsselung, Logging, Rate-Limits usw.
Die Go-App muss nur einfaches HTTP sprechen und ist von außen gar nicht direkt erreichbar.

---

## 1. Go-App (`main.go`)

Kleiner HTTP-Server, der "Hello World" ausliefert:

```go
addr := "127.0.0.1:8081"
http.ListenAndServe(addr, nil)
```

- `127.0.0.1` statt `:8081`: die App lauscht **nur lokal**. So kann niemand
  nginx (und damit TLS) umgehen, indem er Port 8081 direkt anspricht.

```bash
go build -o hello-app .                 # kompiliert nach ./hello-app
nohup ./hello-app > app.log 2>&1 &      # startet im Hintergrund, Log in app.log
curl http://127.0.0.1:8081              # Test: liefert das HTML
```

## 2. Selbstsigniertes SSL-Zertifikat erzeugen

```bash
sudo mkdir -p /etc/nginx/ssl
sudo openssl req -x509 -nodes -newkey rsa:2048 -days 365 \
  -keyout /etc/nginx/ssl/localhost.key \
  -out    /etc/nginx/ssl/localhost.crt \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
sudo chmod 600 /etc/nginx/ssl/localhost.key
```

| Option | Bedeutung |
|---|---|
| `req -x509` | Statt einer Zertifikatsanfrage (CSR) direkt ein selbstsigniertes Zertifikat ausstellen |
| `-nodes` | Private Key **nicht** mit Passphrase verschlüsseln (nginx könnte beim Start sonst nicht nachfragen) |
| `-newkey rsa:2048` | Neuen 2048-Bit-RSA-Schlüssel gleich mit erzeugen |
| `-days 365` | Gültigkeit: 1 Jahr |
| `-keyout` / `-out` | Ablage von privatem Schlüssel / Zertifikat |
| `-subj "/CN=localhost"` | Common Name; vermeidet interaktive Fragen |
| `-addext subjectAltName=...` | Moderne Browser prüfen den Namen nur noch im SAN-Feld, nicht im CN |
| `chmod 600` | Der private Schlüssel darf nur von root gelesen werden |

**Wichtig:** Das Zertifikat ist selbstsigniert, daher zeigen Browser eine Warnung.
Für öffentliche Server nutzt man stattdessen z. B. Let's Encrypt (`certbot`), das
setzt aber eine echte Domain voraus.

## 3. nginx konfigurieren

Datei `/etc/nginx/sites-available/hello-app`:

```nginx
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name localhost;

    ssl_certificate     /etc/nginx/ssl/localhost.crt;
    ssl_certificate_key /etc/nginx/ssl/localhost.key;
    ssl_protocols       TLSv1.2 TLSv1.3;

    location / {
        proxy_pass http://127.0.0.1:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

- `listen 443 ssl` / `[::]:443`: Port 443 mit TLS, für IPv4 und IPv6.
- `ssl_protocols`: nur TLS 1.2/1.3 (ältere Versionen sind unsicher).
- `proxy_pass`: leitet alle Requests an die Go-App weiter.
- `proxy_set_header ...`: gibt der App den Original-Host, die Client-IP und das
  Protokoll (`https`) weiter. Sonst sähe die App nur Anfragen von 127.0.0.1.

Aktivieren (Debian/Ubuntu-Konvention: `sites-available` = vorhanden,
`sites-enabled` = aktiv, per Symlink):

```bash
sudo ln -s /etc/nginx/sites-available/hello-app /etc/nginx/sites-enabled/hello-app
sudo nginx -t                         # Syntax prüfen, IMMER vor dem Reload
sudo systemctl enable --now nginx     # beim Boot starten und jetzt starten
sudo systemctl reload nginx           # Konfig neu laden (ohne Verbindungen zu kappen)
```

## 4. Firewall (ufw) einrichten

Auf dem System war keine Firewall installiert:

```bash
sudo apt-get install -y ufw
sudo ufw default deny incoming        # alles Eingehende blocken, außer erlaubt
sudo ufw default allow outgoing       # Ausgehendes bleibt erlaubt
sudo ufw allow 5901/tcp comment 'VNC'          # VOR dem Aktivieren! Sonst sperrt man sich aus
sudo ufw allow 443/tcp comment 'HTTPS nginx'   # unser Ziel
sudo ufw enable                       # aktivieren (auch beim Boot)
sudo ufw status verbose               # Kontrolle
```

- Nachträglich freigegeben: `sudo ufw allow 22/tcp comment 'SSH'`. Zuvor lief kein SSH-Server,
  daher installiert und gestartet: `sudo apt-get install -y openssh-server` und
  `sudo systemctl enable --now ssh` (Start sofort + beim Boot). Kontrolle:
  `systemctl is-active ssh`, `ss -tlnp | grep ':22 '`.
- Reihenfolge ist wichtig: erst Regeln für bestehende Zugänge (hier VNC auf 5901),
  dann `enable`. Wäre SSH im Einsatz, bräuchte es analog `ufw allow 22/tcp`.
- Port 8081 wird bewusst **nicht** freigegeben, er ist nur für nginx lokal gedacht.
- Port 80 ist ebenfalls zu: es gibt keinen HTTP→HTTPS-Redirect. Bei Bedarf
  `ufw allow 80/tcp` plus einen `server { listen 80; return 301 https://$host$request_uri; }`.

## 5. Testen

```bash
curl -k https://localhost/            # -k: Zertifikatsprüfung überspringen (selbstsigniert)
sudo ufw status                       # 443/tcp ALLOW IN
ss -tlnp | grep -E ':(443|8081)'      # wer lauscht wo?
```

Ergebnis: `Hello World`, HTTP 200.

Hinweis zu `curl --cacert /etc/nginx/ssl/localhost.crt`: das schlägt mit
OpenSSL-Fehler 32 ("key usage does not include certificate signing") fehl, weil
das Zertifikat nicht als CA markiert ist. Für einen Test genügt `-k`; im Browser
einfach die Warnung bestätigen. Ein korrektes Vertrauen erreicht man mit einer
eigenen CA (z. B. mit `mkcert`) oder Let's Encrypt.
Von **außen** erreichbar ist der Server unter `https://<öffentliche-IP>/`; dann
zeigt der Browser wegen `CN=localhost` zusätzlich eine Namens-Warnung. Für echten
Betrieb braucht es eine Domain und ein passendes Zertifikat.

## 6. Dieses Tutorial als Webseite (`/tutorial`)

Die Go-App rendert diese Markdown-Datei als HTML und liefert sie unter
`https://localhost/tutorial` aus. `/` zeigt weiterhin "Hello World".

```bash
go get github.com/yuin/goldmark      # Markdown-Renderer (mit GFM: Tabellen, Checkboxen)
go build -o hello-app .
pkill hello-app; nohup ./hello-app > app.log 2>&1 &   # App neu starten
curl -k https://localhost/tutorial
```

- `goldmark` wandelt Markdown in HTML um; eine kleine HTML-Vorlage mit CSS
  (inkl. Dark-Mode) umrahmt das Ergebnis.
- Die Datei `NGINX_SETUP.md` wird bei **jedem Request neu gelesen**: Textänderungen
  sind sofort sichtbar, ohne Neustart. Nur Änderungen an `main.go` brauchen Rebuild + Neustart.
- Die App liest die Datei relativ zum Arbeitsverzeichnis, sie muss also im
  Projektordner gestartet werden.
- nginx und Firewall brauchen keine Anpassung, der Proxy leitet alle Pfade durch.

## Nützliche Befehle im Alltag

```bash
sudo tail -f /var/log/nginx/error.log /var/log/nginx/access.log
sudo systemctl status nginx
tail -f app.log                       # Log der Go-App
pkill hello-app                       # App stoppen
sudo ufw delete allow 443/tcp         # Freigabe wieder entfernen
```
