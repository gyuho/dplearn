import {
  Component,
} from "@angular/core";

import {
  BackendService,
} from "../request.service";

@Component({
  providers: [BackendService],
  selector: "app",
  styleUrls: ["word-predict.component.css"],
  templateUrl: "word-predict.component.html",
})
export class WordPredictComponent {
  public endpoint = "word-predict-request";
  constructor(public backendService: BackendService) {
    backendService.endpoint = this.endpoint;
  }
}
