# Roadmap

## Próximas entregas

### v1.1 — Multi-cloud Storage Support
Abstrair o cliente de storage em uma interface `StorageProvider`, permitindo que o provedor seja selecionado via variável de ambiente (`STORAGE_PROVIDER`).

- [ ] Definir interface `StorageProvider` em `internal/storage/provider.go`
- [ ] Migrar GCS para implementar a interface
- [ ] Adicionar suporte a **AWS S3** (autenticação via IAM Role ou Access Key)
- [ ] Adicionar suporte a **Azure Blob Storage** (autenticação via Managed Identity ou Service Principal)
- [ ] Factory de provider em `main.go` baseado em env var

---

### v1.2 — Testes de Integração
Cobertura de testes end-to-end sem dependência de cloud real.

- [ ] Testes de integração para S3 usando **LocalStack**
- [ ] Testes de integração para Azure usando **Azurite**
- [ ] Mock de `StorageProvider` para testes unitários isolados

---

### v1.3 — Observabilidade
Tornar a aplicação observável em ambientes de produção.

- [ ] Métricas via **Prometheus** (duração do download, status do upload, tentativas de retry)
- [ ] Tracing distribuído com **OpenTelemetry**
- [ ] Saída de logs em formato **JSON** configurável via env var

---

### v1.4 — Resiliência e Retry
Melhorar a robustez do processo de download e upload.

- [ ] Retry com backoff exponencial no download do GCS/S3/Azure
- [ ] Retry com backoff exponencial no upload para o MAAS
- [ ] Timeout configurável por operação via env var
- [ ] Suporte a retomada de download parcial (range requests)

---

### v1.5 — CI/CD
Pipeline de integração contínua completo.

- [ ] GitHub Actions: lint, test, build em cada PR
- [ ] Build e push da imagem Docker para registry configurável
- [ ] Geração automática de release notes via Conventional Commits
