{{- if .HasDependencies}}
FROM composer:2 AS deps
WORKDIR /app
COPY composer.json .
RUN composer install --no-dev --optimize-autoloader
{{- end}}

FROM {{.BaseImage}}

RUN apk add --no-cache \
    icu-dev \
    libzip-dev \
    linux-headers \
    libpng-dev \
    libjpeg-turbo-dev \
    freetype-dev \
    postgresql-dev \
    && docker-php-ext-configure gd --with-freetype --with-jpeg \
    && docker-php-ext-install \
    intl \
    zip \
    bcmath \
    sockets \
    pcntl \
    pdo_mysql \
    pdo_pgsql \
    gd

WORKDIR /app
{{- if .HasDependencies}}
COPY --from=deps /app/vendor ./vendor
{{- end}}
COPY server.php handler.php ./
EXPOSE {{.Port}}
CMD ["php", "server.php"]
