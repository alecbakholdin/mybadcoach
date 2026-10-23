# Run the full-stack smoke against the compose network.
#
# The base image bundles the browsers for the pinned @playwright/test version,
# so the suite needs no install step at run time.
FROM mcr.microsoft.com/playwright:v1.64.1-noble

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY playwright.config.ts ./
COPY tests ./tests

CMD ["npx", "playwright", "test"]
