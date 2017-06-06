import { Routes, RouterModule } from '@angular/router';

import { AppComponent } from './app.component';
import { HomeComponent } from './home/home.component';
import { WordPredictComponent } from './word-predict/word-predict.component';
import { NotFoundComponent } from './not-found.component';

const appRoutes: Routes = [
    { path: '', redirectTo: '/home', pathMatch: 'full' },
    { path: 'home', component: HomeComponent },

    // { path: '', redirectTo: '/word-predict', pathMatch: 'full' },
    { path: 'word-predict', component: WordPredictComponent },

    { path: '**', component: NotFoundComponent },
];

export const routing = RouterModule.forRoot(appRoutes);

export const routedComponents = [
    HomeComponent,
    WordPredictComponent,
    NotFoundComponent,
];
