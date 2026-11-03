# Nginx Web Server Setup

## Goal
Serve a health page with Nginx. Load balancers and uptime monitors often probe a
URL like this and expect a fixed answer, here the single word `ok`.

## What you need to do
1. Create `/var/www/html/index.html` containing exactly one line: `ok`
2. Make sure the Nginx configuration is valid
3. Start Nginx
4. Confirm that `http://127.0.0.1/` returns HTTP 200 with the body `ok`

> Nginx is already installed. The lab has no internet access, so `apt-get install` will not work
> and you don't need it.

## Hints

### Create the page
```bash
echo ok > /var/www/html/index.html
cat /var/www/html/index.html
```

### Check the configuration, then start Nginx
```bash
nginx -t
service nginx start      # or simply: nginx
```

### Test it
```bash
curl -i http://127.0.0.1/
```

## What the checker verifies
1. `/var/www/html/index.html` exists
2. the file contains exactly the line `ok` (no other text)
3. `nginx -t` succeeds
4. an `nginx` process is running
5. `http://127.0.0.1/` answers `200` with the body `ok`

## Useful commands
- `ls -l /var/www/html`: list the document root
- `pgrep -a nginx`: show nginx processes
- `ss -ltnp`: show listening ports
- `tail /var/log/nginx/error.log`: why Nginx refused to start
