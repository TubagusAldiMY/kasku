// KasKu CI/CD — Jenkins pipeline
// Topologi: Jenkins @ amd64 → cross-build arm64 (buildx+qemu) → push GHCR → SSH deploy ke VPS ARM.
// Meniru logika .github/workflows/deploy.yml, tapi:
//   - build multi-arch untuk linux/arm64 (target VPS Neoverse-N1)
//   - .env TIDAK di-generate di sini; disimpan permanen di VPS (secrets tak lewat CI)
//   - Go services: context = kasku-backend (butuh shared observability-go), -f <svc>/Dockerfile
//   - Rust services: context = kasku-backend/<svc>
// ponytail: pakai `latest` tag + immutable SHA tag; compose default pull `latest`.

pipeline {
  agent any

  options {
    timestamps()
    disableConcurrentBuilds()
    timeout(time: 90, unit: 'MINUTES')   // qemu arm64 build lambat; beri ruang
  }

  environment {
    REGISTRY   = 'ghcr.io'
    OWNER      = 'tubagusaldimy'                 // GHCR namespace (huruf kecil)
    PLATFORM   = 'linux/arm64'
    SHA        = "${env.GIT_COMMIT ?: 'latest'}"
    // Kredensial (dibuat di Tahap 3)
    GHCR       = credentials('ghcr-token')       // username + PAT (write:packages)
    DEPLOY_SSH = 'vps-ssh'                        // SSH private key ke VPS
    VPS        = 'ubuntu@tubsamy-instance'        // host deploy
    DEPLOY_PATH = '/home/ubuntu/kasku'            // dir deploy di VPS (samakan dgn .env di sana)
    // Frontend butuh URL API saat build (di-bake ke bundle)
    API_BASE_URL = 'https://api.tubsamy.dev/v1'   // sesuaikan dgn domain produksimu
  }

  stages {
    stage('Prep buildx') {
      steps {
        sh '''
          set -e
          docker run --privileged --rm tonistiigi/binfmt --install arm64 >/dev/null 2>&1 || true
          docker buildx inspect kasku >/dev/null 2>&1 || docker buildx create --name kasku --driver docker-container --bootstrap
          docker buildx use kasku
          echo "${GHCR_PSW}" | docker login ${REGISTRY} -u "${GHCR_USR}" --password-stdin
        '''
      }
    }

    stage('Build & push Go services') {
      steps {
        dir('kasku-backend') {
          sh '''
            set -e
            GO_SERVICES="api-gateway auth-service user-service billing-service finance-service transaction-service notification-service investment-service admin-service"
            for svc in $GO_SERVICES; do
              IMAGE="${REGISTRY}/${OWNER}/kasku-${svc}"
              echo "=== build ${svc} (arm64) ==="
              docker buildx build --platform "${PLATFORM}" \
                -f "${svc}/Dockerfile" \
                -t "${IMAGE}:latest" -t "${IMAGE}:${SHA}" \
                --push .
            done
          '''
        }
      }
    }

    stage('Build & push Rust services') {
      steps {
        dir('kasku-backend') {
          sh '''
            set -e
            for svc in price-service sync-service; do
              IMAGE="${REGISTRY}/${OWNER}/kasku-${svc}"
              echo "=== build ${svc} (arm64) ==="
              docker buildx build --platform "${PLATFORM}" \
                -t "${IMAGE}:latest" -t "${IMAGE}:${SHA}" \
                --push "${svc}"
            done
          '''
        }
      }
    }

    stage('Build & push Frontend') {
      steps {
        sh '''
          set -e
          IMAGE="${REGISTRY}/${OWNER}/kasku-frontend"
          docker buildx build --platform "${PLATFORM}" \
            --build-arg "PUBLIC_API_BASE_URL=${API_BASE_URL}" \
            -t "${IMAGE}:latest" -t "${IMAGE}:${SHA}" \
            --push kasku-frontend
        '''
      }
    }

    stage('Deploy ke VPS') {
      steps {
        sshagent(credentials: [env.DEPLOY_SSH]) {
          sh '''
            set -e
            # Sinkronkan compose + infra (BUKAN .env — .env permanen di VPS)
            ssh -o StrictHostKeyChecking=accept-new ${VPS} "mkdir -p ${DEPLOY_PATH}/infra"
            scp kasku-backend/docker-compose.yml ${VPS}:${DEPLOY_PATH}/docker-compose.yml
            scp -r kasku-backend/infra/postgres ${VPS}:${DEPLOY_PATH}/infra/
            scp -r kasku-backend/infra/rabbitmq ${VPS}:${DEPLOY_PATH}/infra/

            # Pull image arm64 terbaru & rolling restart, lalu tunggu healthy
            ssh ${VPS} "bash -s" << 'REMOTE'
              set -e
              cd ${HOME}/kasku
              docker compose -f docker-compose.yml pull
              docker compose -f docker-compose.yml up -d --remove-orphans
              echo "Menunggu service healthy..."
              SERVICES="kasku-frontend:3000 kasku-api-gateway:8080 kasku-auth-service:8081 kasku-user-service:8082 kasku-billing-service:8083 kasku-finance-service:8084 kasku-transaction-service:8085 kasku-investment-service:8086 kasku-price-service:8087 kasku-sync-service:8088 kasku-notification-service:8089 kasku-admin-service:8090"
              START=$(date +%s); TIMEOUT=240
              for sp in $SERVICES; do
                C="${sp%%:*}"; P="${sp##*:}"; printf "  %s..." "$C"
                until docker exec "$C" wget -qO- "http://localhost:${P}/health" >/dev/null 2>&1; do
                  if [ $(( $(date +%s) - START )) -gt $TIMEOUT ]; then
                    echo " TIMEOUT"; docker compose ps; docker compose logs --tail=50; exit 1
                  fi
                  sleep 3
                done
                echo " OK"
              done
              echo "Deploy selesai. Semua service healthy."
REMOTE
          '''
        }
      }
    }
  }

  post {
    success { echo "✅ Deploy sukses: ${SHA}" }
    failure { echo "❌ Pipeline gagal — cek log stage di atas." }
    always  { sh 'docker logout ${REGISTRY} || true' }
  }
}
