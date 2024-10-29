# Open Telemetry + GoLang StatsD Demo

This project demonstrates a simple Go application with StatsD metrics and Open Telemetry Collector,
all orchestrated using Docker Compose. The application generates its own load and reports metrics, which are
then collected by OTel Collector and sent to an Oodle endpoint.

## Prerequisites

- Docker
- Docker Compose
- The application requires 6767, 8125 and 8888 ports to be free.

## Project Structure

- `app.go`: The main Go application that generates metrics.
- `Dockerfile`: Used to build the Go application container.
- `docker-compose.yml`: Defines and configures the services.
- `otel-collector-config.yaml`: Configuration file for OTel Collector.

## Setup

1. Clone this repository:
   ```shell
   git clone https://github.com/oodle-ai/otel-go-statsd-demo.git
   cd otel-go-statsd-demo
   ```

2. Create a `.env` file in the `otel-go-stats-demo` directory with the following content
   by replacing placeholders with your account-specific details:
   ```shell
   OODLE_API_KEY=<API_KEY>
   OODLE_OTEL_METRICS_ENDPOINT=https://<OODLE_COLLECTOR_ENDPOINT>/v1/otlp/metrics/<OODLE_INSTANCE>
   OODLE_PROMETHEUS_ENDPOINT=https://<OODLE_COLLECTOR_ENDPOINT>/v1/prometheus/<OODLE_INSTANCE>/write
   ```

## Running the Application

1. Start the services:
   ```shell
   docker-compose up --build
   ```

   This command will build the Go application and start all services defined in the `docker-compose.yml` file.

2. The services will be available at the following addresses:
   - Go Application: http://localhost:6767

3. On successful launch, metrics will be available for consumption in your Oodle UI.

## Stopping the Application

To stop the application and remove the containers, use:

```shell
docker-compose down
```

## Troubleshooting

If you encounter any issues:

1. Ensure all required ports (6767, 8125 and 8888) are free on your host machine.
   If you want to change the ports to be used, you can update ports used in relevant files.
2. Check the Docker logs for any error messages:
   ```
   docker-compose logs
   ```
3. Verify that your API key is correctly set in the `.env` file.
4. Make sure the Oodle endpoint is accessible and correctly configured.