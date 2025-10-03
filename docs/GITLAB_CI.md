# GitLab CI/CD Pipeline para Chaos Mesh Plugin

## 🚀 Visão Geral

O pipeline do GitLab CI foi configurado para automatizar o build, teste e publicação do plugin Chaos Mesh para Argo Rollouts.

## 📋 Stages do Pipeline

### 1. **Test Stage**
- Executa testes unitários
- Gera relatório de cobertura
- Valida a qualidade do código
- Roda em: `main`, `develop`, `merge_requests`, `tags`

### 2. **Build Stage**
- **build**: Build simples para Linux AMD64 (todas as branches)
- **build-multiarch**: Build para múltiplas plataformas (apenas tags)

### 3. **Publish Stage**
- **publish-storage**: Publica binário principal no GCS
- **publish-release**: Publica release multiarch (apenas tags)

## 🔧 Variáveis de Ambiente

```yaml
variables:
  GO_VERSION: "1.21"
  BINARY_NAME: "chaos-mesh-plugin"
  VERSION: "${CI_COMMIT_TAG:-v0.1.0-${CI_COMMIT_SHORT_SHA}}"
```

## 📦 Artefatos Gerados

### Build Simples (todas as branches)
```
chaos-mesh-plugin              # Binário Linux AMD64
chaos-mesh-plugin.sha256       # Checksum SHA256
```

### Build Multiarch (apenas tags)
```
dist/
├── chaos-mesh-plugin-linux-amd64
├── chaos-mesh-plugin-linux-amd64.sha256
├── chaos-mesh-plugin-linux-arm64
├── chaos-mesh-plugin-linux-arm64.sha256
├── chaos-mesh-plugin-darwin-amd64
├── chaos-mesh-plugin-darwin-amd64.sha256
├── chaos-mesh-plugin-darwin-arm64
├── chaos-mesh-plugin-darwin-arm64.sha256
├── chaos-mesh-plugin-windows-amd64.exe
└── chaos-mesh-plugin-windows-amd64.exe.sha256
```

## 🏷️ Versionamento

### Branches
- **main/develop**: `v0.1.0-{commit-sha}`
- **merge_requests**: `v0.1.0-{commit-sha}`

### Tags
- **v1.0.0**: `v1.0.0`
- **v1.2.3-beta**: `v1.2.3-beta`

## 📤 Publicação no Google Cloud Storage

### Estrutura no GCS
```
gs://argo-rollouts-plugin-hml/
├── chaos-mesh-plugin                    # Latest build
├── chaos-mesh-plugin-latest             # Latest release
├── chaos-mesh-plugin-v1.0.0             # Versioned binary
├── chaos-mesh-plugin-v1.0.0.sha256      # Checksum
└── releases/
    └── v1.0.0/
        ├── chaos-mesh-plugin-linux-amd64
        ├── chaos-mesh-plugin-linux-amd64.sha256
        ├── chaos-mesh-plugin-linux-arm64
        ├── chaos-mesh-plugin-linux-arm64.sha256
        ├── chaos-mesh-plugin-darwin-amd64
        ├── chaos-mesh-plugin-darwin-amd64.sha256
        ├── chaos-mesh-plugin-darwin-arm64
        ├── chaos-mesh-plugin-darwin-arm64.sha256
        ├── chaos-mesh-plugin-windows-amd64.exe
        └── chaos-mesh-plugin-windows-amd64.exe.sha256
```

## 🚀 Como Usar

### 1. Push para Branch
```bash
git push origin main
```
- ✅ Executa testes
- ✅ Build Linux AMD64
- ✅ Publica no GCS

### 2. Criar Release
```bash
git tag v1.0.0
git push origin v1.0.0
```
- ✅ Executa testes
- ✅ Build multiarch
- ✅ Publica release completa

### 3. Merge Request
```bash
git push origin feature-branch
# Criar MR no GitLab
```
- ✅ Executa testes
- ✅ Build Linux AMD64
- ❌ Não publica

## 🔍 Monitoramento

### Logs do Pipeline
- Acesse: `GitLab > Projeto > CI/CD > Pipelines`
- Clique no pipeline desejado
- Visualize logs de cada job

### Artefatos
- Acesse: `GitLab > Projeto > CI/CD > Jobs`
- Clique no job de build
- Baixe os artefatos na seção "Job artifacts"

### Coverage Report
- Acesse: `GitLab > Projeto > CI/CD > Pipelines`
- Visualize a porcentagem de cobertura
- Baixe o relatório HTML dos artefatos

## 🛠️ Troubleshooting

### Build Falha
1. Verifique os logs do job `build`
2. Confirme que `go.mod` está correto
3. Verifique se todas as dependências estão disponíveis

### Testes Falham
1. Execute localmente: `go test -v ./...`
2. Corrija os testes que falharam
3. Faça commit e push

### Publicação Falha
1. Verifique credenciais do GCS
2. Confirme que o bucket existe
3. Verifique permissões de escrita

## 🔐 Configuração Necessária

### Variáveis do GitLab CI
Configure em: `GitLab > Projeto > Settings > CI/CD > Variables`

```
GOOGLE_APPLICATION_CREDENTIALS  # Credenciais do GCS (se necessário)
```

### Service Account (GCS)
- Crie um service account no Google Cloud
- Conceda permissões de escrita no bucket
- Configure as credenciais no GitLab

## 📈 Melhorias Futuras

- [ ] Adicionar testes de integração
- [ ] Implementar security scanning
- [ ] Adicionar notificações Slack
- [ ] Implementar deploy automático
- [ ] Adicionar cache de dependências Go
- [ ] Implementar rollback automático

## 🎯 Comandos Úteis

### Testar Localmente
```bash
# Executar testes
go test -v ./...

# Build local
make build

# Build multiarch
make build-all

# Simular pipeline
gitlab-runner exec docker test
gitlab-runner exec docker build
```

### Download do Binário
```bash
# Latest
gsutil cp gs://argo-rollouts-plugin-hml/chaos-mesh-plugin ./

# Versão específica
gsutil cp gs://argo-rollouts-plugin-hml/chaos-mesh-plugin-v1.0.0 ./

# Release multiarch
gsutil cp gs://argo-rollouts-plugin-hml/releases/v1.0.0/chaos-mesh-plugin-linux-amd64 ./
```