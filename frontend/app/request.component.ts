import {
  Injectable,
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

// Request represents TypeScript version of Request in https://github.com/gyuho/deephardway/blob/master/backend/web/web.go.
export class Request {
  public data_from_frontend: string;
  public cancel_request: boolean;
  constructor(
    d: string,
    cancel: boolean,
  ) {
    this.data_from_frontend = d;
    this.cancel_request = cancel;
  }
}

// Item represents TypeScript version of Item in https://github.com/gyuho/deephardway/blob/master/pkg/etcd-queue/queue.go.
export class Item {
  public bucket: string;
  public created_at: string;
  public key: string;
  public value: string;
  public progress: number;
  public canceled: boolean;
  public error: string;
  public request_id: string;
  constructor(
    bucket: string,
    key: string,
    value: string,
    progress: number,
    error: string,
    reqID: string,
  ) {
    this.bucket = bucket;
    this.created_at = "";
    this.key = key;
    this.value = value;
    this.progress = progress;
    this.canceled = false;
    this.error = error;
    this.request_id = reqID;
  }
}

@Injectable()
export class BackendService implements OnDestroy {
  public endpoint = "";

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
    this.inputValue = "";
    this.srespError = "";
    this.result = "No results to show yet...";
  }

  public ngOnDestroy() {
    console.log("user left page!");
    clearInterval(this.pollingHandler);

    const creq = new Request(this.inputValue, true);
    let responseFromSubscribe: Item;
    this.cancelRequest(creq).subscribe(
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

  public cancelRequest(creq: Request): Observable<Item> {
    creq.cancel_request = true;
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
    this.pollingHandler = setInterval(() => this.processRequest(), 1000);
  }
}
