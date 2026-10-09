# ValaDeploy Demo — Angular + Go + PostgreSQL

Projet de validation Universal Build et stack multi-services.

- `frontend/` : Angular 20 (compilation Angular CLI, sans Dockerfile)
- `backend/` : Go HTTP API (sans Dockerfile)
- `database` : PostgreSQL (image standard pour les tests locaux)
- `.local/compose.yaml` : environnement Docker **local uniquement**, ignoré par Git. Ne pas pousser ce fichier pour éviter de masquer la détection Universal Build.

## Test local sans installation de Go/Node

Depuis le dossier racine :

```bash
docker compose -f .local/compose.yaml up -d
docker compose -f .local/compose.yaml logs -f --tail=40 backend frontend
```

Frontend : http://127.0.0.1:14200
Backend : http://127.0.0.1:18480/api/v1/health

Tester l'API :

```bash
curl -fsS http://127.0.0.1:18480/api/v1/health
curl -fsS http://127.0.0.1:18480/api/v1/tasks
curl -fsS -X POST http://127.0.0.1:18480/api/v1/tasks -H 'Content-Type: application/json' -d '{"title":"Test de persistance ValaDeploy"}'
curl -fsS http://127.0.0.1:18480/api/v1/tasks
```

La première installation `npm install` et le téléchargement des images peuvent prendre du temps.

Pour vérifier le build Angular après le lancement :

```bash
docker compose -f .local/compose.yaml exec -T frontend npm run build
```

L'API Go a 5 tests unitaires lancés automatiquement avant son démarrage dans le conteneur. Le stockage persiste dans le volume Docker `postgres-data`.

Pour arrêter **sans supprimer les données** :

```bash
docker compose -f .local/compose.yaml down
```

Attention : ne pas faire `down -v` pendant les tests de persistance.

## Déploiement ValaDeploy

Après validation locale, pousser uniquement `frontend/`, `backend/`, `.gitignore`, `README.md` (+ fichiers lock créés par npm/Go) sur GitHub. Pas de Dockerfile, pas de compose.yaml suivi par Git. Le backend exige `DATABASE_URL` (PostgreSQL). Les modalités de provisioning du service DB dans ValaDeploy devront être contrôlées dans son interface avant le test E2E.

Angular calcule l'API publique à partir de son hostname :
`https://<slug>-frontend.valadeploy.internal` → `https://<slug>-backend.valadeploy.internal/api/v1`.
