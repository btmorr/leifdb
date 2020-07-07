FROM node:14-alpine

RUN npm install express && \
  npm install axios && \
  npm install http-proxy-middleware

ADD build /usr/local/ui/build
ADD server.js /usr/local/ui/server.js
WORKDIR /usr/local/ui

CMD ["node", "server.js"]