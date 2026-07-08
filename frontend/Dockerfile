FROM node:24-alpine AS build

WORKDIR /app

COPY frontend/package*.json ./
RUN npm ci

COPY frontend .
RUN npm run build

FROM nginx:1.27-alpine

COPY deployments/nginx/frontend.conf.template /etc/nginx/templates/default.conf.template
COPY deployments/nginx/docker-entrypoint-with-resolver.sh /usr/local/bin/docker-entrypoint-with-resolver.sh
COPY --from=build /app/dist /usr/share/nginx/html
RUN chmod +x /usr/local/bin/docker-entrypoint-with-resolver.sh

ENTRYPOINT ["/usr/local/bin/docker-entrypoint-with-resolver.sh"]
CMD ["nginx", "-g", "daemon off;"]

EXPOSE 80
