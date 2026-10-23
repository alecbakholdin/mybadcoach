# Build the SvelteKit client from the repo-root workspace, then serve the
# production build with Vite's preview server.
FROM node:22.22-alpine AS build

WORKDIR /app

# Install the workspaces with the committed lockfile.
COPY package.json package-lock.json ./
COPY packages/shared/package.json ./packages/shared/
COPY apps/client/package.json ./apps/client/
RUN npm ci

COPY tsconfig.base.json ./
COPY packages/shared ./packages/shared
COPY apps/client ./apps/client
RUN npm run build -w apps/client

FROM node:22.22-alpine

WORKDIR /app
COPY --from=build /app /app

EXPOSE 3000
CMD ["npm", "run", "preview", "-w", "apps/client", "--", "--host", "0.0.0.0", "--port", "3000"]
