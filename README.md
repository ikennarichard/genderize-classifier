# Genderize Classifier

A fast, lightweight Go API that extends the [Genderize.io](https://genderize.io) API.

## Features

- Single GET endpoint for name gender classification
- Returns structured response with confidence score

## API Endpoint

### Classify Name

```bash
GET /api/classify?name={name}
```

#### Success Response (200 OK)

```bash
{
  "status": "success",
  "data": {
    "name": "john",
    "gender": "male",
    "probability": 0.99,
    "sample_size": 1234,
    "is_confident": true,
    "processed_at": "2026-04-10T17:39:00Z"
  }

```

## Local Development

### Clone repository

```bash
git clone https://github.com/ikennarichard/genderize-classifier
cd genderize-classifier
```

### Run the server

```bash
go run .
```

### Test locally

```bash
curl "http://localhost:8080/api/classify?name=richard
```

## Deployment

This project is configured for easy deployment on Vercel.
