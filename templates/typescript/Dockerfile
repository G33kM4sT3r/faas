FROM {{.BaseImage}}
WORKDIR /app
COPY server.ts .
{{- if .HasDependencies}}
COPY package.json .
RUN bun install
{{- end}}
EXPOSE {{.Port}}
CMD ["bun", "run", "server.ts"]
