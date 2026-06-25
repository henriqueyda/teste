"""OpenTelemetry setup for the agent service.

Configures a TracerProvider (OTLP HTTP → Jaeger) with W3C TraceContext propagation
so MCPClient.call_tool() can inject traceparent headers, creating a continuous trace
across Python → Go MCP. Token counts are recorded as span attributes (not metrics)
because Jaeger all-in-one has no OTLP metrics receiver.
"""
from __future__ import annotations

from opentelemetry import propagate, trace
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import SERVICE_NAME, Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

_SERVICE = "banking-agent"


def setup_telemetry(otlp_endpoint: str) -> None:
    resource = Resource({SERVICE_NAME: _SERVICE})

    tp = TracerProvider(resource=resource)
    tp.add_span_processor(BatchSpanProcessor(
        OTLPSpanExporter(endpoint=f"{otlp_endpoint}/v1/traces")
    ))
    trace.set_tracer_provider(tp)

    propagate.set_global_textmap(TraceContextTextMapPropagator())
