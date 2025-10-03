# 🚀 GitLab CI/CD - Configuração Completa

## ✅ O que foi configurado

Seu pipeline GitLab CI/CD está **100% configurado e pronto para uso**! Aqui está o resumo:

## 📋 Pipeline Stages

### 1. **Test** 🧪
- Executa `go test -v ./...`
- Gera relatório de cobertura
- Roda em: branches, MRs, tags

### 2. **Build** 🔨
- **build**: Linux AMD64 (todas as branches)
- **build-multiarch**: Múltiplas plataformas (apenas tags)

### 3. **Publish** 📤
- **publish-storage**: GCS upload (main + tags)
- **publish-release**: Release multiarch (apenas tags)

## 🎯 Como usar na sua pipeline

### 1. **Push normal (desenvolvimento)**
```bash
git push origin main
```
**Resultado:**
- ✅ Testes executados
- ✅ Build Linux AMD64
- ✅ Upload para GCS: `gs://argo-rollouts-plugin-hml/chaos-mesh-plugin`

### 2. **Release (produção)**
```bash
git tag v1.0.0
git push origin v1.0.0
```
**Resultado:**
- ✅ Testes executados
- ✅ Build multiarch (Linux, macOS, Windows)
- ✅ Upload completo para GCS com versionamento

## 📦 Artefatos gerados

### Build Normal
```
chaos-mesh-plugin              # Binário Linux AMD64
chaos-mesh-plugin.sha256       # Checksum
```

### Build Release (Tag)
```
dist/chaos-mesh-plugin-linux-amd64
dist/chaos-mesh-plugin-linux-arm64
dist/chaos-mesh-plugin-darwin-amd64
dist/chaos-mesh-plugin-darwin-arm64
dist/chaos-mesh-plugin-windows-amd64.exe
+ checksums SHA256 para todos
```

## 🌐 Estrutura no Google Cloud Storage

```
gs://argo-rollouts-plugin-hml/
├── chaos-mesh-plugin                    # Latest build
├── chaos-mesh-plugin-latest             # Latest release
├── chaos-mesh-plugin-v1.0.0             # Versioned
├── chaos-mesh-plugin-v1.0.0.sha256      # Checksum
└── releases/v1.0.0/                     # Full release
    ├── chaos-mesh-plugin-linux-amd64
    ├── chaos-mesh-plugin-linux-arm64
    ├── chaos-mesh-plugin-darwin-amd64
    ├── chaos-mesh-plugin-darwin-arm64
    └── chaos-mesh-plugin-windows-amd64.exe
```

## 🔧 Variáveis configuradas

```yaml
GO_VERSION: "1.21"                        # Versão do Go
BINARY_NAME: "chaos-mesh-plugin"          # Nome do binário
VERSION: "${CI_COMMIT_TAG:-v0.1.0-${CI_COMMIT_SHORT_SHA}}"  # Versionamento automático
```

## 🎉 Funcionalidades implementadas

- ✅ **Versionamento automático**: Tags viram versões, commits viram dev builds
- ✅ **Build multiarch**: Linux, macOS, Windows (AMD64 + ARM64)
- ✅ **Checksums SHA256**: Para validação de integridade
- ✅ **Coverage reporting**: Integrado ao GitLab
- ✅ **Artefatos organizados**: Download fácil pelo GitLab
- ✅ **GCS publishing**: Upload automático para seu bucket
- ✅ **Version flag**: `./chaos-mesh-plugin --version`

## 🚀 Próximos passos

### 1. **Teste o pipeline**
```bash
# Faça um push para testar
git push origin main

# Ou crie uma tag para testar release
git tag v0.1.0
git push origin v0.1.0
```

### 2. **Monitore no GitLab**
- Vá em: `Projeto > CI/CD > Pipelines`
- Acompanhe a execução
- Baixe os artefatos

### 3. **Verifique no GCS**
```bash
# Liste os arquivos
gsutil ls gs://argo-rollouts-plugin-hml/

# Baixe o binário
gsutil cp gs://argo-rollouts-plugin-hml/chaos-mesh-plugin ./
```

## 🔍 Comandos úteis

### Download direto do GCS
```bash
# Latest
curl -L https://storage.googleapis.com/argo-rollouts-plugin-hml/chaos-mesh-plugin -o chaos-mesh-plugin

# Versão específica
curl -L https://storage.googleapis.com/argo-rollouts-plugin-hml/chaos-mesh-plugin-v1.0.0 -o chaos-mesh-plugin
```

### Verificar integridade
```bash
# Baixar checksum
curl -L https://storage.googleapis.com/argo-rollouts-plugin-hml/chaos-mesh-plugin-v1.0.0.sha256 -o checksum

# Verificar
sha256sum -c checksum
```

## 🎯 Está tudo pronto!

Seu pipeline GitLab CI/CD está **completamente configurado** e pronto para:

1. ✅ **Testar** automaticamente todo código
2. ✅ **Buildar** para múltiplas plataformas
3. ✅ **Publicar** no Google Cloud Storage
4. ✅ **Versionar** automaticamente
5. ✅ **Gerar** checksums de segurança

**Basta fazer push e o pipeline fará todo o trabalho!** 🚀

---

📚 **Documentação completa**: `docs/GITLAB_CI.md`