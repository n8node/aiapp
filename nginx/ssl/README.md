# Production TLS files. Do not commit .pem files.

Copy Let's Encrypt material here before `make prod`:

```
fullchain.pem
privkey.pem
```

Use `scripts/issue-certs.sh` on the server. Do not request a certificate for `bitrix.rigintel.ai` — that host is a separate Bitrix24 site.
