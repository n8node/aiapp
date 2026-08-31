# syntax=docker/dockerfile:1

FROM python:3.12-slim-bookworm
WORKDIR /app
RUN useradd --create-home --uid 65532 --shell /usr/sbin/nologin nonroot \
    && pip install --no-cache-dir --upgrade pip
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY app ./app
USER 65532:65532
ENV PYTHONUNBUFFERED=1
EXPOSE 8091
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8091"]
