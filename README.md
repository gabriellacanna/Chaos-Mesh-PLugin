# Argo Rollouts Chaos Mesh Plugin

Um plugin de métrica para o Argo Rollouts que integra nativamente com o Chaos Mesh para executar experimentos de caos durante deployments canary.

## Visão Geral

Este plugin permite que o Argo Rollouts execute experimentos de caos do Chaos Mesh como parte do processo de análise durante rollouts canary. O plugin:

- Cria experimentos de caos dinamicamente no Chaos Mesh
- Injeta seletores de labels automaticamente para atingir apenas pods do ReplicaSet experiment
- Monitora a execução do experimento até conclusão
- Reporta sucesso/falha para o Argo Rollouts determinar se o rollout deve prosseguir

## Funcionalidades

- ✅ **Integração Nativa**: Funciona como um AnalysisProvider do Argo Rollouts
- ✅ **Seleção Dinâmica**: Identifica automaticamente pods do ReplicaSet experiment usando labels
- ✅ **Suporte Multi-Chaos**: Suporta todos os tipos de experimentos do Chaos Mesh (PodChaos, NetworkChaos, etc.)
- ✅ **Monitoramento em Tempo Real**: Acompanha o status do experimento até conclusão
- ✅ **Cleanup Automático**: Opção de limpar experimentos após execução
- ✅ **Timeout Configurável**: Controle de timeout para experimentos
- ✅ **Logging Detalhado**: Logs estruturados para debugging

## Arquitetura

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Argo Rollouts │    │  Chaos Mesh      │    │   Target Pods   │
│                 │    │  Plugin          │    │                 │
│ AnalysisRun ────┼───▶│                  │    │ ReplicaSet      │
│                 │    │ 1. Parse Config  │    │ (experiment)    │
│ AnalysisTemplate│    │ 2. Create Chaos  │    │                 │
│                 │    │ 3. Inject Labels ├───▶│ Chaos Target    │
│ Rollout         │    │ 4. Monitor       │    │                 │
│                 │◀───┤ 5. Report Result │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Instalação

### Método 1: Executável Local

1. Baixe o binário da release:
```bash
curl -L -o chaos-mesh-plugin https://github.com/gabriellacanna/chaos-mesh-plugin/releases/download/v0.1.0/chaos-mesh-plugin-linux-amd64
chmod +x chaos-mesh-plugin
```

2. Configure o Argo Rollouts:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argo-rollouts-config
  namespace: argo-rollouts
data:
  metricProviderPlugins: |-
    - name: "argo-rollouts-chaos-mesh-plugin"
      location: "file://./chaos-mesh-plugin"
```

### Método 2: HTTP/HTTPS

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argo-rollouts-config
  namespace: argo-rollouts
data:
  metricProviderPlugins: |-
    - name: "argo-rollouts-chaos-mesh-plugin"
      location: "https://github.com/gabriellacanna/chaos-mesh-plugin/releases/download/v0.1.0/chaos-mesh-plugin-linux-amd64"
      sha256: "your-sha256-checksum-here"
```

### Método 3: Container Sidecar

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: argo-rollouts
spec:
  template:
    spec:
      containers:
      - name: argo-rollouts
        image: quay.io/argoproj/argo-rollouts:stable
        # ... outras configurações
      - name: chaos-mesh-plugin
        image: gabriellacanna/chaos-mesh-plugin:v0.1.0
        volumeMounts:
        - name: plugin-volume
          mountPath: /plugins
      volumes:
      - name: plugin-volume
        emptyDir: {}
```

## Configuração

### AnalysisTemplate

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: chaos-mesh-experiment-analysis
spec:
  args:
    - name: replica-set-hash
    - name: chaos-spec
  metrics:
    - name: chaos-mesh-test
      provider:
        plugin:
          argo-rollouts-chaos-mesh-plugin:
            # Configuração do experimento de caos (YAML como string)
            chaosExperimentCRD: "{{args.chaos-spec}}"
            # Label para identificar o ReplicaSet target
            targetReplicaSetLabel: "rollouts-pod-template-hash"
            # Valor da label (injetado dinamicamente pelo Argo Rollouts)
            targetReplicaSetValue: "{{args.replica-set-hash}}"
            # Timeout para o experimento (opcional, padrão: 5m)
            timeout: "10m"
            # Limpar experimento após conclusão (opcional, padrão: true)
            cleanupOnFinish: true
```

### Rollout com Experimento de Caos

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: my-app
spec:
  replicas: 5
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: my-app
        image: my-app:v1.0.0
        ports:
        - containerPort: 8080
  strategy:
    canary:
      steps:
      # Passo 1: Deploy com 0% de tráfego + Traffic Mirroring
      - setWeight: 0
      - experiment:
          duration: 5m
          templates:
          - name: chaos-experiment
            specRef: stable
          analyses:
          - name: run-chaos-test
            templateName: chaos-mesh-experiment-analysis
            args:
            # Argo Rollouts injeta automaticamente o hash do novo ReplicaSet
            - name: replica-set-hash
              value: "{{rollouts.new.podTemplateHash}}"
            # Definição do experimento de caos
            - name: chaos-spec
              value: |
                apiVersion: chaos-mesh.org/v1alpha1
                kind: PodChaos
                metadata:
                  name: pod-kill-experiment-{{rollouts.new.podTemplateHash}}
                  namespace: default
                spec:
                  action: pod-kill
                  mode: one
                  # O plugin irá injetar o selector automaticamente
                  selector:
                    namespaces:
                      - default
                  scheduler:
                    cron: "@every 30s"
                  duration: "2m"
      
      # Passo 2: Se o caos passou, inicia rollout real
      - setWeight: 25
      - pause: { duration: 2m }
      - setWeight: 50
      - pause: { duration: 2m }
      - setWeight: 75
      - pause: { duration: 2m }
      - setWeight: 100
```

## Exemplos de Experimentos

### PodChaos - Matar Pods

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: PodChaos
metadata:
  name: pod-kill-test
  namespace: default
spec:
  action: pod-kill
  mode: one
  selector:
    namespaces:
      - default
    # labelSelectors será injetado automaticamente pelo plugin
  duration: "60s"
```

### NetworkChaos - Latência de Rede

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: NetworkChaos
metadata:
  name: network-delay-test
  namespace: default
spec:
  action: delay
  mode: all
  selector:
    namespaces:
      - default
    # labelSelectors será injetado automaticamente pelo plugin
  delay:
    latency: "100ms"
    correlation: "100"
    jitter: "10ms"
  duration: "2m"
```

### StressChaos - Stress de CPU

```yaml
apiVersion: chaos-mesh.org/v1alpha1
kind: StressChaos
metadata:
  name: cpu-stress-test
  namespace: default
spec:
  mode: one
  selector:
    namespaces:
      - default
    # labelSelectors será injetado automaticamente pelo plugin
  stressors:
    cpu:
      workers: 2
      load: 80
  duration: "3m"
```

## Parâmetros de Configuração

| Parâmetro | Tipo | Obrigatório | Descrição |
|-----------|------|-------------|-----------|
| `chaosExperimentCRD` | string | ✅ | YAML do experimento Chaos Mesh |
| `targetReplicaSetLabel` | string | ✅ | Nome da label para identificar ReplicaSet |
| `targetReplicaSetValue` | string | ✅ | Valor da label do ReplicaSet target |
| `chaosMeshEndpoint` | string | ❌ | URL da API do Chaos Mesh (usa in-cluster por padrão) |
| `timeout` | string | ❌ | Timeout do experimento (padrão: "5m") |
| `cleanupOnFinish` | bool | ❌ | Limpar experimento após conclusão (padrão: true) |

## Fluxo de Execução

1. **Inicialização**: Plugin recebe configuração do AnalysisTemplate
2. **Validação**: Valida parâmetros obrigatórios
3. **Parse do CRD**: Faz parse do YAML do experimento de caos
4. **Injeção de Seletor**: Injeta `labelSelectors` com o hash do ReplicaSet
5. **Criação**: Cria o experimento no Chaos Mesh via API Kubernetes
6. **Monitoramento**: Observa o status até fase "Finished" ou timeout
7. **Resultado**: Reporta sucesso/falha para o Argo Rollouts
8. **Cleanup**: Remove experimento se `cleanupOnFinish=true`

## Troubleshooting

### Plugin não encontrado
```
Error: plugin not found: argo-rollouts-chaos-mesh-plugin
```
**Solução**: Verifique se o plugin está configurado corretamente no ConfigMap `argo-rollouts-config`.

### Erro de permissões
```
Error: failed to create chaos experiment: forbidden
```
**Solução**: Verifique se o ServiceAccount do Argo Rollouts tem permissões para criar CRDs do Chaos Mesh:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: argo-rollouts-chaos-mesh
rules:
- apiGroups: ["chaos-mesh.org"]
  resources: ["*"]
  verbs: ["*"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: argo-rollouts-chaos-mesh
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: argo-rollouts-chaos-mesh
subjects:
- kind: ServiceAccount
  name: argo-rollouts
  namespace: argo-rollouts
```

### Timeout do experimento
```
Error: timeout waiting for experiment to complete
```
**Solução**: Aumente o valor do parâmetro `timeout` ou verifique se o experimento está sendo executado corretamente no Chaos Mesh.

### Experimento não encontra pods
```
Warning: No pods found matching selector
```
**Solução**: Verifique se:
- O ReplicaSet experiment foi criado corretamente
- A label `rollouts-pod-template-hash` está presente nos pods
- O namespace está correto

## Desenvolvimento

### Pré-requisitos

- Go 1.21+
- Docker
- Kubernetes cluster com Chaos Mesh instalado
- Argo Rollouts instalado

### Build Local

```bash
# Clonar repositório
git clone https://github.com/gabriellacanna/chaos-mesh-plugin.git
cd chaos-mesh-plugin

# Instalar dependências
make deps

# Build
make build

# Executar testes
make test

# Build para múltiplas plataformas
make build-all
```

### Testes

```bash
# Testes unitários
make test

# Testes com coverage
make test-coverage

# Lint
make lint
```

## Contribuição

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/amazing-feature`)
3. Commit suas mudanças (`git commit -m 'Add amazing feature'`)
4. Push para a branch (`git push origin feature/amazing-feature`)
5. Abra um Pull Request

## Licença

Este projeto está licenciado sob a Licença Apache 2.0 - veja o arquivo [LICENSE](LICENSE) para detalhes.

## Suporte

- 📖 [Documentação do Argo Rollouts](https://argo-rollouts.readthedocs.io/)
- 📖 [Documentação do Chaos Mesh](https://chaos-mesh.org/docs/)
- 🐛 [Issues](https://github.com/gabriellacanna/chaos-mesh-plugin/issues)
- 💬 [Discussions](https://github.com/gabriellacanna/chaos-mesh-plugin/discussions)