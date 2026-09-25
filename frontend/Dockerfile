# Etapa 1: build (spec frontend RF-F-01)
FROM node:20-alpine AS build
WORKDIR /app
ARG VITE_API_URL=http://localhost:8080
ENV VITE_API_URL=$VITE_API_URL
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Etapa 2: Caddy sirviendo dist/ en 5173 con fallback SPA
FROM caddy:2-alpine
COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=build /app/dist /srv
EXPOSE 5173
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
