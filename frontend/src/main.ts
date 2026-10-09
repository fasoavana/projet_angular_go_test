import { Component, OnInit, signal, provideZoneChangeDetection } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { bootstrapApplication } from '@angular/platform-browser';

interface Task { id: number; title: string; done: boolean; }
interface Health { status: string; service: string; database: string; }

function apiBase(): string {
  const hostname = window.location.hostname;
  if (hostname.endsWith('-frontend.valadeploy.internal')) {
    const backend = hostname.replace(/-frontend\.valadeploy\.internal$/, '-backend.valadeploy.internal');
    return `https://${backend}/api/v1`;
  }
  return '/api/v1'; // local Angular dev proxy -> Go container
}
const api = apiBase();

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <main class="wrap">
      <div class="eyebrow">VALADEPLOY • DÉMONSTRATION UNIVERSAL BUILD</div>
      <h1>Angular <span>+</span> Go <span>+</span> PostgreSQL</h1>
      <p class="intro">Application de test : un frontend Angular, une API Go et des données persistantes dans PostgreSQL.</p>
      <section class="grid">
        <article class="card">
          <div class="cardlabel">État du backend</div>
          <div class="status" [class.online]="health()?.status === 'online'">{{ health()?.status === 'online' ? '● ONLINE' : '● NON DISPONIBLE' }}</div>
          <div class="meta">Service : {{ health()?.service || '—' }}</div>
          <div class="meta">Base de données : {{ health()?.database || '—' }}</div>
        </article>
        <article class="card">
          <div class="cardlabel">Connectivité</div>
          <div class="status">{{ tasks().length }} tâche(s)</div>
          <div class="meta">Frontend → Go API → PostgreSQL</div>
          <div class="meta">URL API : {{ api }}</div>
        </article>
      </section>
      <section class="card tasks">
        <div class="taskstop">
          <div><h2>Test de persistance</h2><p>Ajoutez une tâche puis redémarrez les conteneurs : la tâche doit rester.</p></div>
          <button class="secondary" type="button" (click)="refresh()">Actualiser</button>
        </div>
        <form (ngSubmit)="addTask()" class="taskform">
          <input name="title" aria-label="Titre de la tâche" maxlength="120" [(ngModel)]="title" placeholder="Ex. Valider SecurityProfile" required>
          <button type="submit" [disabled]="busy() || !title.trim()">{{ busy() ? 'Ajout...' : 'Ajouter' }}</button>
        </form>
        <p class="error" *ngIf="error()">{{ error() }}</p>
        <div class="empty" *ngIf="!tasks().length && !error()">Aucune tâche. Ajoutez-en une pour tester la base.</div>
        <ul><li *ngFor="let task of tasks()"><span class="bullet">✓</span><span>{{ task.title }}</span><span class="id">#{{ task.id }}</span></li></ul>
      </section>
      <footer>Aucun Dockerfile dans le dépôt · Déploiement attendu via Universal Build</footer>
    </main>
  `
})
class AppComponent implements OnInit {
  readonly health = signal<Health | null>(null);
  readonly tasks = signal<Task[]>([]);
  readonly error = signal('');
  readonly busy = signal(false);
  readonly api = api;
  title = '';

  ngOnInit(): void { void this.refresh(); }

  async refresh(): Promise<void> {
    this.error.set('');
    try {
      const [h, t] = await Promise.all([fetch(`${api}/health`), fetch(`${api}/tasks`)]);
      if (!h.ok || !t.ok) throw new Error('API ou base de données indisponible');
      this.health.set(await h.json() as Health);
      this.tasks.set(await t.json() as Task[]);
    } catch (err) {
      this.health.set(null);
      this.error.set(err instanceof Error ? err.message : 'Erreur de connexion');
    }
  }

  async addTask(): Promise<void> {
    const title = this.title.trim();
    if (!title || this.busy()) return;
    this.busy.set(true);
    this.error.set('');
    try {
      const response = await fetch(`${api}/tasks`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ title })
      });
      if (!response.ok) throw new Error(`Ajout refusé (HTTP ${response.status})`);
      this.title = '';
      await this.refresh();
    } catch (err) {
      this.error.set(err instanceof Error ? err.message : 'Erreur de sauvegarde');
    } finally { this.busy.set(false); }
  }
}

bootstrapApplication(AppComponent, { providers: [provideZoneChangeDetection()] })
  .catch(err => console.error(err));
