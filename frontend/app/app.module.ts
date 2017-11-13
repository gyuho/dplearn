import { ApplicationRef, NgModule } from "@angular/core";
import { FormsModule } from "@angular/forms";
import { HttpModule, JsonpModule } from "@angular/http";
import { BrowserModule } from "@angular/platform-browser";

import { BrowserAnimationsModule } from "@angular/platform-browser/animations";

import {
  MatButtonModule,
  MatCardModule,
  MatChipsModule,
  MatInputModule,
  MatMenuModule,
  MatProgressSpinnerModule,
  MatSnackBarModule,
  MatToolbarModule,
} from "@angular/material";

import { AppComponent } from "./app.component";
import { routedComponents, routing } from "./app.routing";

@NgModule({
  declarations: [
    AppComponent,
    routedComponents,
  ],
  entryComponents: [AppComponent],
  imports: [
    BrowserModule,
    FormsModule,

    HttpModule,
    JsonpModule,

    BrowserAnimationsModule,

    MatButtonModule,
    MatToolbarModule,
    MatCardModule,
    MatMenuModule,
    MatInputModule,
    MatSnackBarModule,
    MatProgressSpinnerModule,
    MatChipsModule,

    routing,
  ],
})

export class AppModule {
  constructor(private _appRef: ApplicationRef) { }

  public ngDoBootstrap() {
    this._appRef.bootstrap(AppComponent);
  }
}
