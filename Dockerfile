FROM nginx:alpine

# Copy the SVG file to the Nginx document root
COPY cheezymeme.svg /usr/share/nginx/html/

# Create a simple index.html that displays the SVG
RUN echo '<!DOCTYPE html>' > /usr/share/nginx/html/index.html && \
    echo '<html><head><title>Meme Viewer</title></head>' >> /usr/share/nginx/html/index.html && \
    echo '<body style="margin:0; padding:20px; text-align:center; background:#f0f0f0;">' >> /usr/share/nginx/html/index.html && \
    echo '<h1>Your Meme</h1>' >> /usr/share/nginx/html/index.html && \
    echo '<img src="cheezymeme.svg" alt="Meme" style="max-width:100%; height:auto; border:1px solid #ccc; background:white;">' >> /usr/share/nginx/html/index.html && \
    echo '</body></html>' >> /usr/share/nginx/html/index.html
