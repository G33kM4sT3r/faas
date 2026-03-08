FROM {{.BaseImage}}
WORKDIR /app
COPY server.js .
{{- if .HasDependencies}}
COPY package.json .
RUN bun install
{{- end}}
EXPOSE {{.Port}}
CMD ["bun", "run", "server.js"]
