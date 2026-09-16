# syntax=docker/dockerfile:1.4
#
# metahub-admin-ui — single static Go binary with the SPA embedded
# (go:embed). Multi-arch friendly: stages that EXECUTE commands are pinned
# to --platform=$BUILDPLATFORM so nothing runs under QEMU; the final alpine
# stage is multi-arch.
#
# No private Go modules ⇒ no SSH mount needed (unlike metahub-api/admin).

# --- Stage 1: build the SPA (arch-independent JS) ---------------------------
FROM --platform=$BUILDPLATFORM node:lts-alpine AS ui
WORKDIR /ui
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# --- Stage 2: compile Go with the freshly built SPA embedded ----------------
FROM --platform=$BUILDPLATFORM golang:1.24-alpine3.21 AS go
WORKDIR /go/metahub-admin-ui
COPY . .
# Real build output replaces the committed placeholder before compiling.
COPY --from=ui /ui/dist ./dist
RUN --mount=type=cache,target=/go/pkg/mod CGO_ENABLED=0 go build -ldflags '-s -w' -o metahub-admin-ui .

# --- Stage 3: minimal runtime -----------------------------------------------
FROM alpine:3.21
ENV METAHUB_ADMIN_UI_PORT=8080
COPY --from=go /go/metahub-admin-ui/metahub-admin-ui /usr/bin/metahub-admin-ui
EXPOSE 8080
ENTRYPOINT ["/usr/bin/metahub-admin-ui"]
