import { RouterModule, Routes } from "@angular/router";

import { AppComponent } from "./app.component";
import { HomeComponent } from "./home/home.component";

import { CatsVsDogsComponent } from "./cats-vs-dogs/cats-vs-dogs.component";
import { WordPredictComponent } from "./word-predict/word-predict.component";

import { NotFoundComponent } from "./not-found.component";

const appRoutes: Routes = [
    { path: "", redirectTo: "/home", pathMatch: "full" },
    { path: "home", component: HomeComponent },

    { path: "word-predict", component: WordPredictComponent },
    { path: "cats-vs-dogs", component: CatsVsDogsComponent },

    { path: "**", component: NotFoundComponent },
];

export const routing = RouterModule.forRoot(appRoutes);

export const routedComponents = [
    HomeComponent,
    WordPredictComponent,
    CatsVsDogsComponent,
    NotFoundComponent,
];
