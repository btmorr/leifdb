const express = require('express');
const cors = require('cors');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();

app.use(cors())
app.use(express.json());
app.post('/proxy', function(req, res) {
    console.log("Body:", req.body);
    if (!req.body.address) {
        console.log("no address field in body");
        res.status(400).send("request body must include address field to configure proxy")
    } else {
        console.log(`Proxy address: ${req.body.address}`);
        app.use('/health', createProxyMiddleware({ target: req.body.address, changeOrigin: true }));
        app.use('/db', createProxyMiddleware({ target: req.body.address, changeOrigin: true }));
        res.send(`Proxy set to ${req.body.address}`);
    }
});
console.log("Listening on localhost:4000")
app.listen(4000);
