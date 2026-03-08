FROM {{.BaseImage}}
WORKDIR /app
{{- if .HasDependencies}}
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
{{- end}}
COPY server.py .
EXPOSE {{.Port}}
CMD ["python", "server.py"]
