# Godfather - Business & Political Event Monitoring System

## Overview

Godfather is a comprehensive monitoring system designed to track and analyze business and political events. The system is built using a microservices architecture with Golang backend services, PostgreSQL for data storage, NATS for inter-service communication, and an Angular frontend.

## Features

- **Real-time event monitoring** of business and political activities
- **Customizable alerts** based on user-defined criteria
- **Data visualization** through interactive dashboards
- **Multi-source data integration** from various feeds and APIs
- **User role management** with granular permissions

## Architecture

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Angular   │ ◄──┤    API      │ ◄──┤   NATS      │
│  Frontend   │    │   Gateway   │    │  Message    │
└─────────────┘    └─────────────┘    │   Bus       │
                                      └─────────────┘
                                           ▲
                                           │
┌─────────────┐    ┌─────────────┐    ┌────┴─────┐
│ PostgreSQL  │ ◄──┤  Data       │ ◄──┤  Event   │
│  Database   │    │  Processor  │    │ Collector│
└─────────────┘    └─────────────┘    └──────────┘
```

## Services

1. **API Gateway** - Central entry point for all frontend requests
2. **Event Collector** - Gathers data from various sources
3. **Data Processor** - Analyzes and processes incoming events
4. **Alert Engine** - Generates notifications based on rules
5. **User Service** - Manages authentication and authorization

## Technologies

- **Backend**: Go (Golang)
- **Frontend**: Angular
- **Database**: PostgreSQL
- **Messaging**: NATS
- **API Documentation**: Swagger/OpenAPI
- **Containerization**: Docker (compose)

## Getting Started

### Prerequisites

- Go 1.23+
- PostgreSQL 15+
- NATS server
- Docker (optional)


## License

Distributed under the MIT License. See `LICENSE` for more information.

## Contact

Project Maintainer - [Andrey Sploshnov] - ice.nightcrawler@gmail.com

Project Link - [https://github.com/TuliMyrskyTaivas/godfather](https://github.com/TuliMyrskyTaivas/godfather)
