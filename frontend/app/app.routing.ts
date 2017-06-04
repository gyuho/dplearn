import { Routes, RouterModule } from '@angular/router';

import { AppComponent } from './app.component';
import { HomeComponent } from './home/home.component';
import { SpellCheckComponent } from './spell-check/spell-check.component';
import { NotFoundComponent } from './not-found.component';

const appRoutes: Routes = [
    { path: '', redirectTo: '/home', pathMatch: 'full' },
    { path: 'home', component: HomeComponent },

    // { path: '', redirectTo: '/spell-check', pathMatch: 'full' },
    { path: 'spell-check', component: SpellCheckComponent },

    { path: '**', component: NotFoundComponent },
];

export const routing = RouterModule.forRoot(appRoutes);

export const routedComponents = [
    HomeComponent,
    SpellCheckComponent,
    NotFoundComponent,
];
