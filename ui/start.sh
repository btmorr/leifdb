node /usr/local/proxy/proxy.js &

serve -l tcp://0.0.0.0:3000 -s /usr/local/ui
