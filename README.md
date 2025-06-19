# Helios Sync Server<sup>beta</sup>

The **Helios Sync Server** powers secure, end-to-end encrypted synchronization for [Helios](https://heliosch.at)—the blazing fast, privacy-first LLM chat client. This Go-based backend is designed for speed, simplicity, and zero-knowledge privacy: your data is always encrypted client-side, and the server never sees your secrets.

## 🚀 Features

- **Zero-Knowledge Sync**: All chat data is encrypted in your browser. The server only stores encrypted blobs—no plaintext, no metadata, no user accounts.
- **Stateless Authentication**: Sync with a passphrase—no registration, no email, no tracking.
- **Lightning Fast**: Built with Go and Redis for maximum performance and reliability.
- **Simple API**: Minimal, well-documented endpoints for easy integration with the Helios frontend.
- **Open Source**: MIT licensed and ready for your contributions.

## 🛠️ Getting Started

1. **Clone the repository**

   ```sh
   git clone https://github.com/helioschat/helios-sync.git
   cd helios-sync
   ```

2. **Configure environment**

   Copy or edit the `.env` file to set your configuration (see example values in `.env.example`).

3. **Run with Docker (recommended)**

   ```sh
   docker build -t helios-sync .
   docker run -p 8080:8080 --env-file .env helios-sync
   ```

   Or run locally:

   ```sh
   go run main.go
   ```

4. **Connect your frontend**

   Point your Helios frontend to your sync server’s URL.

## 📄 License

MIT License. See [LICENSE](./LICENSE) for details.
