from langchain_google_genai import ChatGoogleGenerativeAI
from .config import settings


def get_model() -> ChatGoogleGenerativeAI:
    if not settings.google_api_key:
        raise RuntimeError("GOOGLE_API_KEY is required")
    return ChatGoogleGenerativeAI(
        model=settings.google_model,
        google_api_key=settings.google_api_key,
    )
