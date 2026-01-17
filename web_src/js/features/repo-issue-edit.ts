import {handleReply} from './repo-issue.ts';
import {addDelegatedEventListener} from '../utils/dom.ts';
import {getComboMarkdownEditor, initComboMarkdownEditor, ComboMarkdownEditor} from './comp/ComboMarkdownEditor.ts';
import {POST} from '../modules/fetch.ts';
import {showErrorToast} from '../modules/toast.ts';
import {hideElem, querySingleVisibleElem, showElem} from '../utils/dom.ts';
import {triggerUploadStateChanged} from './comp/EditorUpload.ts';
import {convertHtmlToMarkdown} from '../markup/html2markdown.ts';
import {applyAreYouSure, reinitializeAreYouSure} from '../vendor/jquery.are-you-sure.ts';

async function tryOnEditContent(e: Event) {
  const clickTarget = (e.target as HTMLElement).closest('.edit-content');
  if (!clickTarget) return;

  e.preventDefault();

  // If the menu item is outside the comment DOM (dropdown appended to body), it should include a data-target attribute
  // pointing to the raw content element (eg. "issuecomment-<id>-raw"). Prefer using that when present.
  let commentContent: HTMLElement;
  const dataTarget = (clickTarget as HTMLElement).getAttribute('data-target');
  if (dataTarget) {
    const raw = document.querySelector<HTMLElement>(`#${dataTarget}`);
    if (!raw) return;
    commentContent = raw.parentElement as HTMLElement;
  } else {
    commentContent = clickTarget.closest('.comment-header')!.nextElementSibling as HTMLElement;
  }

  const editContentZone = commentContent.querySelector('.edit-content-zone')!;
  let renderContent = commentContent.querySelector('.render-content')!;
  const rawContent = commentContent.querySelector('.raw-content')!;

  let comboMarkdownEditor : ComboMarkdownEditor;

  const cancelAndReset = (e: Event) => {
    e.preventDefault();
    showElem(renderContent);
    hideElem(editContentZone);
    comboMarkdownEditor.dropzoneReloadFiles();
  };

  const saveAndRefresh = async (e: Event) => {
    e.preventDefault();
    // we are already in a form, do not bubble up to the document otherwise there will be other "form submit handlers"
    // at the moment, the form submit event conflicts with initRepoDiffConversationForm (global '.conversation-holder form' event handler)
    e.stopPropagation();
    renderContent.classList.add('is-loading');
    showElem(renderContent);
    hideElem(editContentZone);
    try {
      const params = new URLSearchParams({
        content: comboMarkdownEditor.value(),
        context: String(editContentZone.getAttribute('data-context')),
        content_version: String(editContentZone.getAttribute('data-content-version')),
      });
      for (const file of comboMarkdownEditor.dropzoneGetFiles() ?? []) {
        params.append('files[]', file);
      }

      const response = await POST(editContentZone.getAttribute('data-update-url')!, {data: params});
      const data = await response.json();
      if (!response.ok) {
        showErrorToast(data?.errorMessage ?? window.config.i18n.error_occurred);
        return;
      }

      reinitializeAreYouSure(editContentZone.querySelector('form')); // the form is no longer dirty
      editContentZone.setAttribute('data-content-version', data.contentVersion);

      // replace the render content with new one, to trigger re-initialization of all features
      const newRenderContent = renderContent.cloneNode(false) as HTMLElement;
      newRenderContent.innerHTML = data.content;
      renderContent.replaceWith(newRenderContent);
      renderContent = newRenderContent;

      rawContent.textContent = comboMarkdownEditor.value();

      if (!commentContent.querySelector('.dropzone-attachments')) {
        if (data.attachments !== '') {
          commentContent.insertAdjacentHTML('beforeend', data.attachments);
        }
      } else if (data.attachments === '') {
        commentContent.querySelector('.dropzone-attachments')!.remove();
      } else {
        commentContent.querySelector('.dropzone-attachments')!.outerHTML = data.attachments;
      }
      comboMarkdownEditor.dropzoneSubmitReload();
    } catch (error) {
      showErrorToast(`Failed to save the content: ${error}`);
      console.error(error);
    } finally {
      renderContent.classList.remove('is-loading');
    }
  };

  // Show write/preview tab and copy raw content as needed
  showElem(editContentZone);
  hideElem(renderContent);

  comboMarkdownEditor = getComboMarkdownEditor(editContentZone.querySelector('.combo-markdown-editor'))!;
  if (!comboMarkdownEditor) {
    editContentZone.innerHTML = document.querySelector('#issue-comment-editor-template')!.innerHTML;
    const form = editContentZone.querySelector('form')!;
    applyAreYouSure(form);
    const saveButton = querySingleVisibleElem<HTMLButtonElement>(editContentZone, '.ui.primary.button')!;
    const cancelButton = querySingleVisibleElem<HTMLButtonElement>(editContentZone, '.ui.cancel.button')!;
    comboMarkdownEditor = await initComboMarkdownEditor(editContentZone.querySelector('.combo-markdown-editor')!);
    const syncUiState = () => saveButton.disabled = comboMarkdownEditor.isUploading();
    comboMarkdownEditor.container.addEventListener(ComboMarkdownEditor.EventUploadStateChanged, syncUiState);
    cancelButton.addEventListener('click', cancelAndReset);
    form.addEventListener('submit', saveAndRefresh);
  }

  // FIXME: ideally here should reload content and attachment list from backend for existing editor, to avoid losing data
  if (!comboMarkdownEditor.value()) {
    comboMarkdownEditor.value(rawContent.textContent);
  }
  comboMarkdownEditor.switchTabToEditor();
  comboMarkdownEditor.focus();
  triggerUploadStateChanged(comboMarkdownEditor.container);
}

function extractSelectedMarkdown(container: HTMLElement) {
  const selection = window.getSelection();
  if (!selection?.rangeCount) return '';
  const range = selection.getRangeAt(0);
  if (!container.contains(range.commonAncestorContainer)) return '';

  // todo: if commonAncestorContainer parent has "[data-markdown-original-content]" attribute, use the parent's markdown content
  // otherwise, use the selected HTML content and respect all "[data-markdown-original-content]/[data-markdown-generated-content]" attributes
  const contents = selection.getRangeAt(0).cloneContents();
  const el = document.createElement('div');
  el.append(contents);
  return convertHtmlToMarkdown(el);
}

async function tryOnQuoteReply(e: Event) {
  const clickTarget = (e.target as HTMLElement).closest('.quote-reply');
  if (!clickTarget) return;

  e.preventDefault();
  const contentToQuoteId = clickTarget.getAttribute('data-target');
  // If a data-target is available (menu item was rendered with it), prefer that as it works even if menu is moved out of the comment DOM.
  let targetRawToQuote: HTMLElement | null = null;
  if (contentToQuoteId) {
    targetRawToQuote = document.querySelector<HTMLElement>(`#${contentToQuoteId}.raw-content`);
  }
  if (!targetRawToQuote) {
    // fallback to old approach of locating markup inside comment DOM
    const candidate = clickTarget.closest('.comment-header')?.nextElementSibling as HTMLElement | undefined;
    if (candidate) targetRawToQuote = candidate.querySelector<HTMLElement>('.raw-content');
  }
  if (!targetRawToQuote) return;

  const targetMarkupToQuote = targetRawToQuote.parentElement!.querySelector<HTMLElement>('.render-content.markup')!;
  let contentToQuote = extractSelectedMarkdown(targetMarkupToQuote);
  if (!contentToQuote) contentToQuote = targetRawToQuote.textContent;
  const quotedContent = `${contentToQuote.replace(/^/mg, '> ')}\n\n`;

  let editor;
  if (clickTarget.classList.contains('quote-reply-diff')) {
    // Prefer to find the reply button by using the raw content's parent conversation-holder when dropdown is outside the DOM
    const replyBtn = (targetRawToQuote.parentElement!.closest('.conversation-holder') || clickTarget.closest('.comment-code-cloud'))?.querySelector<HTMLElement>('button.comment-form-reply')!;
    editor = await handleReply(replyBtn);
  } else {
    // for normal issue/comment page
    editor = getComboMarkdownEditor(document.querySelector('#comment-form .combo-markdown-editor'))!;
  }

  if (editor.value()) {
    editor.value(`${editor.value()}\n\n${quotedContent}`);
  } else {
    editor.value(quotedContent);
  }
  editor.focus();
  editor.moveCursorToEnd();
}

let _repoIssueCommentEditInited = false;
export function initRepoIssueCommentEdit() {
  if (_repoIssueCommentEditInited) return;
  _repoIssueCommentEditInited = true;

  // Use pointerdown delegation so we catch menu item interactions even if dropdown removes DOM nodes on click
  addDelegatedEventListener(document, 'pointerdown', '.edit-content, .quote-reply', (el: HTMLElement, e: PointerEvent) => {
    e.preventDefault();
    // el is either .edit-content or .quote-reply
    if (el.matches('.edit-content')) tryOnEditContent(e as unknown as Event);
    if (el.matches('.quote-reply') || el.closest('.quote-reply')) tryOnQuoteReply(e as unknown as Event);
  });
}
