# 📚 Protoflow Documentation

Welcome to the knowledge base. This is where we keep the deep dives, the nitty-gritty details, and the advanced techniques that would clutter up the main README.

## 🗺️ The Map

- [**Handlers Guide**](handlers/README.md) 🧠
  - Learn how to write type-safe handlers for Protobuf and JSON.
  - Master metadata manipulation and publishing patterns.
  - Understand how to use the `ServiceLogger` effectively.

- [**Configuration Guide**](configuration/README.md) ⚙️
  - Configure your transports (Kafka, RabbitMQ, AWS).
  - Tweak the middleware stack to your liking.
  - Inject custom dependencies like validators and outbox stores.

- [**Development Guide**](development/README.md) 🛠️
  - Set up your local environment for contributing to Protoflow.
  - Run tests, linters, and examples using `task`.
  - Tips and tricks for debugging local brokers.

## 🔗 Related Resources

- **`examples/`**: Runnable code is worth a thousand words. Check out `simple`, `json`, `proto`, and `full` for working examples.
- **`internal/runtime/`**: The engine room. If you're curious how the magic happens, this is where the source code lives.
