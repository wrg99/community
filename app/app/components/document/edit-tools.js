// Copyright 2016 Documize Inc. <legal@documize.com>. All rights reserved.
//
// This software (Documize Community Edition) is licensed under
// GNU AGPL v3 http://www.gnu.org/licenses/agpl-3.0.en.html
//
// You can operate outside the AGPL restrictions by purchasing
// Documize Enterprise Edition and obtaining a commercial license
// by contacting <sales@documize.com>.
//
// https://documize.com

import Ember from 'ember';
import TooltipMixin from '../../mixins/tooltip';

const {
	inject: { service }
} = Ember;

export default Ember.Component.extend(TooltipMixin, {
	link: service(),
	linkName: '',
	keywords: '',
	selection: null,
	tabs: [
		{ label: 'Section', selected: true },
		{ label: 'Attachment', selected: false },
		{ label: 'Search', selected: false }
	],

	showSections: Ember.computed('tabs.@each.selected', function() {
		return this.get('tabs').findBy('label', 'Section').selected;
	}),
	showAttachments: Ember.computed('tabs.@each.selected', function() {
		return this.get('tabs').findBy('label', 'Attachment').selected;
	}),
	showSearch: Ember.computed('tabs.@each.selected', function() {
		return this.get('tabs').findBy('label', 'Search').selected;
	}),

	init() {
		this._super(...arguments);
		let self = this;

		let folderId = this.get('folder.id');
		let documentId = this.get('document.id');
		let pageId = this.get('page.id');

		this.get('link').getCandidates(folderId, documentId, pageId).then(function (candidates) {
			self.set('candidates', candidates);
			self.set('hasSections', is.not.null(candidates.pages) && candidates.pages.length);
			self.set('hasAttachments', is.not.null(candidates.attachments) && candidates.attachments.length);
		});
	},

	didRender() {
		this.addTooltip(document.getElementById("content-linker-button"));
	},

	willDestroyElement() {
		this.destroyTooltips();
	},

	actions: {
		setSelection(i) {
			this.set('selection', i);

			let candidates = this.get('candidates');

			candidates.pages.forEach(c => {
				Ember.set(c, 'selected', c.id === i.id);
			});

			candidates.attachments.forEach(c => {
				Ember.set(c, 'selected', c.id === i.id);
			});
		},

		onInsertLink() {
			let selection = this.get('selection');

			if (is.null(selection)) {
				return;
			}

			return this.get('onInsertLink')(selection);
		},

		onTabSelect(tabs) {
			this.set('tabs', tabs);
		}
	}
});