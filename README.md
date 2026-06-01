# Raggy: An Architectural Playground for Event-Driven Go Microservices

Raggy is a work-in-progress project I designed to develop my software engineering skills and further my learning with golang and AI tools.

Primary Goals:
- Pure Microservices 
- Event-driven patterns with Apache Kafka
- Retrieval-Augmented Generation (RAG) pipelines

Instead of bundling everything into a single application, this project breaks the system down into decoupled services to explore how microservice architectures handle data isolation, fault tolerance, and cross-service observability.

## Learning Objectives & Architecture Exploration

This repository serves as a practical space for me to step out of my comfort zone and experiment with advanced architectural patterns:

### 1. Event-Driven Architectures 
Instead of building a traditional, synchronous system where services are tightly coupled via REST APIs, I integrated Apache Kafka to explore asynchronous messages. This ensures that slow, heavy data-parsing tasks occur completely out-of-band, preventing data processing spikes from ever blocking user-facing web services or degrading the core user experience.

### 2. Bounded Contexts (Domain-Driven Design) & Scaling Workloads (CPU vs. I/O)
A common trap in microservice design is creating a "distributed monolith" where services are dependent on one another through RPC/HTTP. 

I used this project to practice strict domain isolation, decoupling the I/O-bound services from CPU-bound work. 

- I/O-bound: HTTP services such as dashboard and RAG (query)
- CPU-bound: ingestion, parsing, chunking and other processing of files

This structural boundary allows each microservice to scale horizontally and independently based on its specific compute or storage workload constraints.

### 3. Centralized Logging & Observability
In a distributed system, debugging across multiple independent services is incredibly difficult without consistent telemetry. I used this project to learn how to utilize structured logs for easy aggregation, enabling the use of observability tools in the future. 

### Tools Involved

- Docker
- Golang
- Nginx
- Kafka
- S3 Object Store
- Postgresql Databases (Standard & PgVector)
- React 
- Gemini
