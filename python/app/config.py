from pydantic_settings import BaseSettings, SettingsConfigDict
from pydantic import Field


class Settings(BaseSettings):
    model_config = SettingsConfigDict(env_file=[".env", "../.env"], extra="ignore", populate_by_name=True)

    mcp_server_url: str = Field(default="http://localhost:7070/mcp", alias="MCP_SERVER_URL")
    google_api_key: str = Field(default="", alias="GOOGLE_API_KEY")
    google_model: str = Field(default="gemini-2.0-flash", alias="GOOGLE_MODEL")
    faiss_index_path: str = Field(default="./.faiss_index", alias="FAISS_INDEX_PATH")
    checkpoint_db_url: str = Field(
        default="postgresql://agent_user:agent_user@localhost:5432/bank",
        alias="CHECKPOINT_DB_URL",
    )
    otel_exporter_otlp_endpoint: str = Field(
        default="http://localhost:4318", alias="OTEL_EXPORTER_OTLP_ENDPOINT"
    )


settings = Settings()
