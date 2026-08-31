FROM wordpress:6.7-php8.3-apache

COPY tools/wp-cli.phar /usr/local/bin/wp
RUN chmod +x /usr/local/bin/wp && wp --info --allow-root

COPY conf/forwarded-https.conf /etc/apache2/conf-available/forwarded-https.conf
COPY conf/uploads.ini /usr/local/etc/php/conf.d/uploads.ini
COPY docker-entrypoint-custom.sh /usr/local/bin/docker-entrypoint-custom.sh
RUN a2enconf forwarded-https && \
    chmod +x /usr/local/bin/docker-entrypoint-custom.sh && \
    mkdir -p /var/www/html/wp-content/uploads && \
    chown -R www-data:www-data /var/www/html/wp-content

ENTRYPOINT ["docker-entrypoint-custom.sh"]
CMD ["apache2-foreground"]
