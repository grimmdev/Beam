# Beam ⚡

> **Secure, ephemeral file transfer for your Home Lab.**
> Zero-config, web-based, and fast.

![Go Version](https://img.shields.io/github/go-mod/go-version/grimmdev/beam)
![Latest Release](https://img.shields.io/github/v/release/grimmdev/beam?label=latest)
![Build Status](https://img.shields.io/github/actions/workflow/status/grimmdev/beam/release.yml)
![License](https://img.shields.io/github/license/grimmdev/beam)

Beam is a lightweight tool to transfer files between devices on your local network (LAN). It solves the problem of "how do I get this file from my PC to my phone/laptop/guest without using the cloud or logging into email?"

## ✨ Features

* **Simple PIN System:** No QR codes or long URLs. Just a 4-digit PIN.
* **Burn After Reading:** Optional self-destruct mode deletes the file immediately after one successful download.
* **Auto-Expiry:** Files automatically delete after 10 minutes, 1 hour, or 24 hours.
* **Single Binary:** Written in Go, compiles to a single executable (or tiny Docker image).
* **Privacy First:** Files stay on your LAN. No external servers involved.

## 🐳 Using Docker

Beam is available as a lightweight Docker container. It supports **amd64** (Standard PC/Server) and **arm64** (Raspberry Pi/Apple Silicon).

### Quick Start (Docker Run)

Run Beam instantly with a single command. This maps port `3000` on your host to the container.

```bash
docker run -d \
  --name beam \
  -p 3000:3000 \
  -v $(pwd)/beam_data:/app/data \
  -v $(pwd)/beam_uploads:/app/uploads \
  ghcr.io/grimmdev/beam:latest
```
Access the app at `http://localhost:3000`.
