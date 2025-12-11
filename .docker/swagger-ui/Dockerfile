FROM swaggerapi/swagger-ui:latest

COPY docs/schema.yaml /usr/share/nginx/html/openapi.yaml
ENV SWAGGER_JSON_URL=/swagger/openapi.yaml
ENV API_URL=/swagger/openapi.yaml
