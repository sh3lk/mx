# 🌿 mx — A Continuation of Service Weaver

**mx** is a community-maintained fork of the original [Service Weaver](https://github.com/ServiceWeaver/weaver) project by Google, which has been archived and is no longer under active development.

This fork aims to keep the project alive, maintain it, and continue evolving its capabilities for modern distributed systems in Go.

---
## 📦 Installation

Ensure you have Go installed, version 1.24 or higher. Then, run the following to install the weaver command:
```bash
go install github.com/sh3lk/mx/cmd/mx@latest
```
go install installs the weaver command to $GOBIN, which defaults to $HOME/go/bin. Make sure this directory is included in your PATH. You can accomplish this, for example, by adding the following to your .bashrc and running source ~/.bashrc:
```bash
export PATH="$PATH:$HOME/go/bin"
```
---

## 📄 Documentation

Much of the documentation remains compatible with [https://serviceweaver.dev](https://serviceweaver.dev), but updates specific to **mx** will be published here as the project evolves.

---



## 💬 Community and Contributions

We welcome contributions of all kinds! Feel free to:

- Open issues and suggest features
- Submit pull requests
- Help test and improve `mx`

---

## 📝 License

This project is based on the original Service Weaver and follows the same license. See [LICENSE](./LICENSE) for details.

---

> _mx is not affiliated with Google. It is an independent continuation of an open-source project._

