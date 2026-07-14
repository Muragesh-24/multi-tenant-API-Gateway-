# <b>General steps to host a production backend

## Push the code to GitHub 
## Pull the code in the vm
## Run it in the vm by similar to you run it locally
## the use nginx for reverse proxy  --- here you will set custom domain(if any)
## the you will get a http link
## then use the certbot to get the ssl certificate 


# <b>Nginx setup for reverse proxy

````
server {

    # Listen for HTTP requests on port 80
    listen 80;

    # Domain name
    server_name api.muragesh.tech;

    # Match all URLs
    location / {

        # Forward requests to Express
        proxy_pass http://localhost:3000;

        # Forward original host
        proxy_set_header Host $host;

        # Forward client's real IP
        proxy_set_header X-Real-IP $remote_addr;

        # Forward protocol (http/https)
        proxy_set_header X-Forwarded-Proto $scheme;

        # Forward proxy chain
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
````
```
upstream backend_servers {

    server localhost:3001;
    server localhost:3002;
    server localhost:3003;

}
````



````
location /auth {

    proxy_pass http://localhost:8001;

}

location /maps {

    proxy_pass http://localhost:8002;

}
````
````
# Install
sudo apt update
sudo apt install nginx

# Create config
sudo nano /etc/nginx/sites-available/api

# Enable config
sudo ln -s /etc/nginx/sites-available/api /etc/nginx/sites-enabled/

# Validate
sudo nginx -t

# Apply
sudo systemctl reload nginx

# Check
sudo systemctl status nginx

# View logs if needed
sudo tail -f /var/log/nginx/error.log
````