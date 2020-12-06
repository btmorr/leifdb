const express = require('express');
const axios = require('axios');
const path = require('path');
const { createProxyMiddleware } = require('http-proxy-middleware');

const app = express();
let currentProxy = "";


// this works for connecting to a server, but not for redirects--how to also proxy redirects?
function refreshProxy() {
    let proxyFound = false;
    let dbNodes = process.env.LEIFDB_MEMBER_NODES.split(",");
    let i  = dbNodes.length;
    while (i > 0 && !proxyFound && dbNodes) {
        i = i - 1;
        const n = dbNodes.pop();
        axios.get(`http://${n}/health`).then( () => {
            if (n !== currentProxy) {
                // only reconfigure proxy if it's different
                app.use('/db', createProxyMiddleware({target: n, changeOrigin: true}));
                currentProxy = n;
            }
            proxyFound = true;
        }).catch((e) => {
            // console.log(e)
        });
    }
}

// set up proxy on boot
console.log("Initializing proxy");
refreshProxy();
// re-check health and set proxy every 5s
console.log("Starting proxy ticker");
setInterval(refreshProxy, 1000);

app.use(express.json());
app.use(express.static(path.join(__dirname, 'build')));

app.get('/*', function(req, res) {
    res.sendFile(path.join(__dirname, 'build', 'index.html'));
});
console.log("Listening on port 3000")
app.listen(3000);
