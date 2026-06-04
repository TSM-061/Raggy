FROM node:22-alpine

WORKDIR /app

# Copy package blueprints first to leverage Docker layer caching
COPY package.json package-lock.json* ./

# Clean install dependencies
RUN npm install

# The standard Vite development execution command
CMD ["npm", "run", "dev", "--", "--host"]
