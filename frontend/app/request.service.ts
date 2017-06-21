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
  Router,
} from "@angular/router";

import {
  Observable,
} from "rxjs/Rx";

import {
  MdSnackBar,
} from "@angular/material";

// Request represents TypeScript version of Request in https://github.com/gyuho/deephardway/blob/master/backend/web/web.go.
export class Request {
  public data_from_frontend: string;
  public create_request: boolean;
  constructor(
    d: string,
    create: boolean,
  ) {
    this.data_from_frontend = d;
    this.create_request = create;
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

  public inputValue: string;
  public result: string;

  public progress = 0;
  public spinnerColor = "primary";
  public spinnerMode = "indeterminate";

  private errorFromServer = "";
  private mode = "Observable";
  private requestID: string;
  private intervalSet: boolean;
  private pollingHandler;
  private url: string;

  constructor(
    private router: Router,
    private http: Http,
    private snackBar: MdSnackBar,
  ) {
    this.inputValue = "";
    this.result = "Nothing to show yet...";
    this.intervalSet = false;

    this.url = router.url;
  }

  public ngOnDestroy() {
    console.log("user left page; destroying!", this.url);
    this.intervalSet = false;
    clearInterval(this.pollingHandler);

    const body = JSON.stringify(new Request(this.inputValue, false));
    const headers = new Headers({"Content-Type" : "application/json"});
    const options = new RequestOptions({headers});

    // this.http.post().map().catch() returns 'Observable<Item>'
    // which is non-blocking, 'subscribe' is the blocking operation
    let itemFromServer: Item;
    this.http.post(this.endpoint, body, options)
      .map(this.processHTTPResponseClient)
      .catch(this.processHTTPErrorClient)
      .subscribe(
        (resp) => itemFromServer = resp,
        (error) => this.errorFromServer = error as any,
        () => this.processItemFromServer(itemFromServer), // on-complete
      );

    this.inputValue = "";
    this.result = "Nothing to show yet...";
    this.progress = 0;
    this.errorFromServer = "";
    console.log("user left page; destroyed!", this.url);

    return;
  }

  public processItemFromServer(resp: Item) {
    this.result = resp.value;
    this.requestID = resp.request_id;

    // set interval only after first response
    if (!this.intervalSet) {
      this.intervalSet = true;
      this.pollingHandler = setInterval(() => this.fetchStatus(), 500);
    }

    if (resp.error !== "") {
      clearInterval(this.pollingHandler);
      this.result = (this.result === "") ? resp.error : `${resp.value} (${resp.error})`;
    }

    if (resp.canceled === true) {
      clearInterval(this.pollingHandler);
      this.result += " - canceled!";
    }

    this.progress = resp.progress;
    if (this.progress === 100) {
      clearInterval(this.pollingHandler);
    }
  }

  public processHTTPResponseClient(res: Response) {
    return (res.json() as Item) || {};
  }

  public processHTTPErrorClient(error: any) {
    const errMsg = (error.message) ? error.message :
      error.status ? `${error.status} - ${error.statusText}` : "Server error";
    console.error(errMsg);
    this.errorFromServer = errMsg;
    return Observable.throw(errMsg);
  }

  // fetchStatus requests status updates from backend server.
  // Blocks until the item is completed.
  // TODO: time out?
  public fetchStatus() {
    const headers = new Headers({
      "Content-Type" : "application/plain",
      "Request-Id" : this.requestID,
    });
    const options = new RequestOptions({headers});

    let itemFromServer: Item;
    this.http.get(this.endpoint, options)
      .map(this.processHTTPResponseClient)
      .catch(this.processHTTPErrorClient)
      .subscribe(
        (resp) => itemFromServer = resp,
        (error) => this.errorFromServer = error as any,
        () => this.processItemFromServer(itemFromServer), // on-complete
      );
  }

  public clickPOST() {
    this.snackBar.open("Job scheduled! Waiting...", "Requested!", {
      duration: 10000,
    });

    this.progress = 0;
    this.result = `[FRONTEND - ACK] Requested '${this.inputValue}' (request ID: ${this.requestID})`;

    const body = JSON.stringify(new Request(this.inputValue, true));
    const headers = new Headers({"Content-Type" : "application/json"});
    const options = new RequestOptions({headers});

    // this.http.post().map().catch() returns 'Observable<Item>'
    // which is non-blocking, 'subscribe' is the blocking operation
    let itemFromServer: Item;
    this.http.post(this.endpoint, body, options)
      .map(this.processHTTPResponseClient)
      .catch(this.processHTTPErrorClient)
      .subscribe(
        (resp) => itemFromServer = resp,
        (error) => this.errorFromServer = error as any,
        () => this.processItemFromServer(itemFromServer), // on-complete
      );

    // retry in case of network glitches
    // DO NOT DO THIS because this.http.post is asynchronous
    // this.pollingHandler = setInterval(() => this.fetchStatus(), 500);
  }
}
