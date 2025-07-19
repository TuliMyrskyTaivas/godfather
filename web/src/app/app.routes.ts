import { Routes } from '@angular/router';
import { SourcesComponent } from './sources/sources.component';
import { TermsComponent } from './terms/terms.component';

export const routes: Routes = [
    { path: '', pathMatch: 'full', redirectTo: 'sources'},
    { path: 'sources', component : SourcesComponent },
    { path: 'terms', component : TermsComponent }
];
