# test_golang

Kleines Lernprojekt: eine simple Go-Web-App hinter einem nginx-Reverse-Proxy
mit HTTPS und Firewall.

## Was die App macht

| Pfad        | Inhalt                                                      |
|-------------|-------------------------------------------------------------|
| `/`         | "Hello World" mit Link zum Tutorial                         |
| `/tutorial` | `NGINX_SETUP.md`, zur Laufzeit als HTML gerendert (goldmark) |

Die App lauscht nur auf `127.0.0.1:8081`. Von außen erreichbar ist sie
ausschließlich über nginx auf Port 443 (HTTPS).

```
Browser ──HTTPS:443──▶ nginx ──HTTP:8081 (nur localhost)──▶ Go-App
```

## Voraussetzungen

- Go (siehe `go.mod`)
- Für den Proxy-Betrieb: nginx, openssl, ufw (siehe [NGINX_SETUP.md](NGINX_SETUP.md))

## Starten

```bash
go build -o hello-app .
./hello-app                  # im Projektordner starten
```

Test:

```bash
curl http://127.0.0.1:8081/            # direkt an die App
curl -k https://localhost/tutorial     # über nginx (selbstsigniertes Zertifikat, daher -k)
```

Die App liest `NGINX_SETUP.md` relativ zum Arbeitsverzeichnis und bei jedem
Request neu. Textänderungen sind also ohne Neustart sichtbar. Änderungen an
`main.go` erfordern Rebuild und Neustart.

## nginx, HTTPS und Firewall

Die komplette, kommentierte Schritt-für-Schritt-Anleitung (Zertifikat erzeugen,
nginx konfigurieren, ufw-Regeln) steht in [NGINX_SETUP.md](NGINX_SETUP.md).

## Projektstruktur

```
.
├── main.go          # Webserver und Markdown-Rendering
├── go.mod / go.sum  # Go-Module und Abhängigkeiten
├── NGINX_SETUP.md   # Tutorial / Protokoll des nginx-Setups
└── README.md
```

## Hinweis zur Sicherheit

Das Zertifikat ist selbstsigniert (`CN=localhost`). Browser zeigen daher eine
Warnung. Für einen öffentlichen Betrieb braucht es eine Domain und ein
Zertifikat einer echten CA, z. B. Let's Encrypt.
