# Security

Unishare is intended for private self-hosted deployments. Run it behind HTTPS and keep all tokens in `UNISHARE_USERS` secret.

## Reporting

If you find a vulnerability, please open a private security advisory on GitHub or contact the repository owner directly.

## Notes

- File links require an authenticated browser cookie.
- Named tokens are isolated server-side; do not reuse the same token for multiple users.
- Android profile or Private Space isolation is not bypassed directly; all clients communicate through the server-side buffer.
- Back up the `/data` volume if the shared buffer contains important files.
