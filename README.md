# Go Forum

A modern, lightweight forum built with Go, Gin, and SQLite. Features a clean interface, user management, and comprehensive moderation tools.

## Installation

First, clone the project:
```bash
git clone https://github.com/birabittoh/go-forum
cd go-forum
cp .env.example .env
```
Then, edit `.env` with your settings.

### Docker
Just run:
```
docker compose up -d
```

### Local

Install dependencies:
```bash
go mod tidy
```

Then run the forum:
```bash
go run .
```

## Email Setup

For Gmail:
1. Enable 2-factor authentication
2. Generate an App Password
3. Use the App Password in `SMTP_PASSWORD`

## Roadmap

- [ ] Search functionality
- [ ] User reputation system
- [ ] Multi-language support
- [ ] Themes
- [ ] File attachments?

## License

This project is open source and available under the [MIT License](LICENSE).
