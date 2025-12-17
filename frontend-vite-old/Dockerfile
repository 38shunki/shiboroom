FROM node:20-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy source code
COPY . .

EXPOSE 5176

CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0", "--port", "5176"]
