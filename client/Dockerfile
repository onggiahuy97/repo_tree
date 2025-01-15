# Use the official PHP Apache image
FROM php:8.2-apache

# Copy everything from the current directory into /var/www/html in the container
COPY . /var/www/html/

# By default, this Apache image listens on port 80
EXPOSE 80

# (Optionally) enable any Apache/PHP modules if needed
# RUN docker-php-ext-install etc.

# The default CMD in php:apache is to start Apache

