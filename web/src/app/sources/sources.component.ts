import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { SourcesService, Source } from '../sources.service';

@Component({
  selector: 'app-sources',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './sources.component.html',
  styleUrl: './sources.component.scss'
})

export class SourcesComponent implements OnInit {
  sources: Source[] = [];

  constructor (private sourcesService: SourcesService) {}
  ngOnInit(): void {
    this.getSources();
  }

  getSources() {
    this.sourcesService.getSources().subscribe((data: Source[]) => {
      this.sources = data
    });
  }

  addSource() {
    var newSource : Source = {
      id: 0,
      name: '',
      uri: '',
      updateInterval: '',
      lastUpdated: ''
    };
    this.sourcesService.addSource(newSource).subscribe(() => {
      this.getSources();
    });
  }

  deleteSource(source: Source) {
    this.sourcesService.deleteSource(source).subscribe(() => {
      this.getSources();
    });
  }
}
