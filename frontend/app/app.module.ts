import { ApplicationRef, NgModule } from "@angular/core";
import { FormsModule } from "@angular/forms";
import { HttpModule, JsonpModule } from "@angular/http";
import { BrowserModule } from "@angular/platform-browser";

import { BrowserAnimationsModule } from "@angular/platform-browser/animations";

import {
  MdButtonModule,
  MdCardModule,
  MdChipsModule,
  MdInputModule,
  MdMenuModule,
  MdProgressSpinnerModule,
  MdSnackBarModule,
  MdToolbarModule,
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

    MdButtonModule,
    MdToolbarModule,
    MdCardModule,
    MdMenuModule,
    MdInputModule,
    MdSnackBarModule,
    MdProgressSpinnerModule,
    MdChipsModule,

    routing,
  ],
})

export class AppModule {
  constructor(private _appRef: ApplicationRef) { }

  public ngDoBootstrap() {
    this._appRef.bootstrap(AppComponent);
  }
}
