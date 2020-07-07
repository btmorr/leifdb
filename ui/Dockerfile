FROM node:14-alpine

RUN npm install -g serve

ADD build /usr/local/ui

CMD ["serve", "-l", "tcp://0.0.0.0:3000", "-s", "/usr/local/ui"]
