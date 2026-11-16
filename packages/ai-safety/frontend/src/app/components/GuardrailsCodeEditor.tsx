import React from 'react';
import { CodeEditor, Language } from '@patternfly/react-code-editor';
import { loader } from '@monaco-editor/react';
import * as monaco from 'monaco-editor';

// Point the monaco loader at the locally bundled monaco instead of the CDN,
// and expose it on window so tests can read the editor model.
loader.config({ monaco });
Object.assign(window, { monaco });

type GuardrailsCodeEditorProps = {
  code: string;
  onCodeChange: (value: string) => void;
  height?: string;
  isReadOnly?: boolean;
  'data-testid'?: string;
};

/**
 * A YAML code editor for guardrail template configuration, backed by the
 * locally bundled monaco. The PF CodeEditor does not forward arbitrary
 * data attributes to its root, so the test id sits on this wrapper.
 */
const GuardrailsCodeEditor: React.FC<GuardrailsCodeEditorProps> = ({
  code,
  onCodeChange,
  height = '300px',
  isReadOnly = false,
  'data-testid': dataTestId,
}) => (
  <div data-testid={dataTestId}>
    <CodeEditor
      isUploadEnabled
      isDownloadEnabled
      isCopyEnabled
      isLanguageLabelVisible
      language={Language.yaml}
      height={height}
      code={code}
      onCodeChange={onCodeChange}
      isReadOnly={isReadOnly}
    />
  </div>
);

export default GuardrailsCodeEditor;
