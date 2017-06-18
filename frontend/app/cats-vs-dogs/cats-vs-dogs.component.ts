import {
  Component,
  OnDestroy,
} from "@angular/core";

import {
  Headers,
  Http,
  RequestOptions,
  Response,
} from "@angular/http";

import {
  Observable,
} from "rxjs/Rx";

import {
  MdSnackBar,
} from "@angular/material";

import {
  Item,
  Request,
} from "../request-item.component";

@Component({
  selector: "app",
  styleUrls: ["cats-vs-dogs.component.css"],
  templateUrl: "cats-vs-dogs.component.html",
})
export class CatsVsDogsComponent implements OnDestroy {
  public endpoint = "cats-vs-dogs-request";

  public mode = "Observable";

  public inputValue: string;

  public sresp: Item;
  public srespError: string;
  public result: string;

  public progress = 0;
  public spinnerColor = "primary";
  public spinnerMode = "indeterminate";

  private pollingHandler;

  constructor(private http: Http, public snackBar: MdSnackBar) {
    this.inputValue = "https://images.pexels.com/photos/127028/pexels-photo-127028.jpeg?w=1260&h=750&auto=compress&cs=tinysrgb";
    this.srespError = "";
    this.result = "No results to show yet...";
  }

  public ngOnDestroy() {
    clearInterval(this.pollingHandler);

    const creq = new Request(this.inputValue, true);
    let responseFromSubscribe: Item;
    this.deleteRequest(creq).subscribe(
      (sresp) => responseFromSubscribe = sresp,
      (error) => this.srespError = error as any,
      () => this.processItem(responseFromSubscribe), // on-complete
    );

    this.inputValue = "";
    this.srespError = "";
    return;
  }

  public processItem(resp: Item) {
    this.sresp = resp;
    this.result = resp.value;
    if (resp.error !== "") {
      clearInterval(this.pollingHandler);
      this.result = (this.result === "") ? resp.error : `${resp.value} (${resp.error})`;
    }
    if (resp.canceled === true) {
      this.result += " - canceled!";
    }

    this.progress = resp.progress;
    if (this.progress === 100) {
      clearInterval(this.pollingHandler);
    }
  }

  public processHTTPResponseClient(res: Response) {
    const jsonBody = res.json();
    const sresp = jsonBody as Item;
    return sresp || {};
  }

  public processHTTPErrorClient(error: any) {
    const errMsg = (error.message) ? error.message :
      error.status ? `${error.status} - ${error.statusText}` : "Server error";
    console.error(errMsg);
    this.srespError = errMsg;
    return Observable.throw(errMsg);
  }

  public postRequest(creq: Request): Observable<Item> {
    const body = JSON.stringify(creq);
    const headers = new Headers({"Content-Type" : "application/json"});
    const options = new RequestOptions({headers});

    // this returns without waiting for POST response
    const obser = this.http.post(this.endpoint, body, options)
      .map(this.processHTTPResponseClient)
      .catch(this.processHTTPErrorClient);
    return obser;
  }

  public deleteRequest(creq: Request): Observable<Item> {
    creq.delete_request = true;
    return this.postRequest(creq);
  }

  public processRequest() {
    const creq = new Request(this.inputValue, false);
    let responseFromSubscribe: Item;
    this.postRequest(creq).subscribe(
      (sresp) => responseFromSubscribe = sresp,
      (error) => this.srespError = error as any,
      () => this.processItem(responseFromSubscribe), // on-complete
    );
  }

  public clickProcessRequest() {
    this.snackBar.open("Job scheduled! Waiting...", "Requested!", {
      duration: 7000,
    });
    this.progress = 0;
    this.pollingHandler = setInterval(() => this.processRequest(), 1500);
  }
}
