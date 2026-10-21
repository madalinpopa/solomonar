# syntax=docker/dockerfile:1
FROM node:24-alpine@sha256:6a1c190e17215d844926b0677f4bf87273ff7d5e54db06d3b5b5863ab0b09a3d AS build
WORKDIR /app/site
COPY site/package.json site/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --ignore-scripts --no-audit --no-fund
COPY site/ ./
COPY docs/logo/ /app/docs/logo/
COPY docs/images/ /app/docs/images/
ARG SITE_ORIGIN=https://solomonar.dev
ARG SITE_BASE_PATH=/
ARG SITE_VERSION=dev
ENV SITE_ORIGIN=${SITE_ORIGIN} SITE_BASE_PATH=${SITE_BASE_PATH} SITE_VERSION=${SITE_VERSION}
RUN npm run build && npm run check

FROM nginx:stable-alpine@sha256:970364228b1e4f932abc22d321932816782cca8a20b85a77552a7cfe9acfb3b5
LABEL org.opencontainers.image.source="https://github.com/madalinpopa/solomonar" \
      org.opencontainers.image.title="Solomonar documentation" \
      org.opencontainers.image.licenses="MIT"
COPY site/nginx.conf /etc/nginx/nginx.conf
COPY --from=build /app/site/_site/ /usr/share/nginx/html/
USER nginx
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O /dev/null http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["nginx"]
CMD ["-g", "daemon off;"]
