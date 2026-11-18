import React from 'react';
import {
  Alert,
  Button,
  Bullseye,
  Form,
  FormGroup,
  FormHelperText,
  HelperText,
  HelperTextItem,
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  Spinner,
  TextInput,
} from '@patternfly/react-core';
import { useNotification } from '~/app/hooks/useNotification';
import { createGuardrailTemplate, updateGuardrailTemplate } from '~/app/api/guardrails';
import { useGuardrailTemplate } from '~/app/hooks/useGuardrails';
import GuardrailsCodeEditor from '~/app/components/GuardrailsCodeEditor';
import { GuardrailTemplate } from '~/app/types';

const TEMPLATE_NAME_PATTERN = /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/;

// Starter skeleton shown for new templates: a working prompt-injection
// guardrail using a server-provided classifier rail. The model is chosen
// when the template is deployed.
const STARTER_CONFIG_YAML = `# Guardrail template: checks user input for prompt injection
# using a local classifier (no model call needed). The model that allowed
# messages are forwarded to is chosen when deploying the template.
passthrough: true
rails:
  input:
    flows:
      - hf classifier check input $classifier="prompt-injection"
  config:
    hf_classifier:
      prompt-injection:
        engine: local
        model: /app/models/prompt-injection
        task: text-classification
        threshold: 0.5
        blocked_labels:
          - injection
`;

export type ManageTemplateTarget =
  | { mode: 'create' }
  | { mode: 'edit'; template: GuardrailTemplate };

type ManageTemplateModalProps = {
  target: ManageTemplateTarget | null;
  defaultNamespace: string;
  onClose: () => void;
  onSaved: (template: GuardrailTemplate) => void;
};

const hasConfigContent = (value: string): boolean => value.trim() !== '';

const ManageTemplateModal: React.FC<ManageTemplateModalProps> = ({
  target,
  defaultNamespace,
  onClose,
  onSaved,
}) => {
  const notification = useNotification();
  const [name, setName] = React.useState('');
  const [description, setDescription] = React.useState('');
  const [configYAML, setConfigYAML] = React.useState('');
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [error, setError] = React.useState<Error | null>(null);

  const isEdit = target?.mode === 'edit';
  const editTemplate = isEdit ? target.template : null;

  // Fetch the current contents for edit mode.
  const [detail, detailLoaded, detailError] = useGuardrailTemplate(
    editTemplate?.name ?? null,
    editTemplate?.namespace ?? '',
  );

  // Prefill from the template detail once it arrives (edit mode only).
  const prefillApplied = React.useRef(false);
  React.useEffect(() => {
    if (isEdit && detail && detailLoaded && !prefillApplied.current) {
      prefillApplied.current = true;
      const configFile = detail.files.find((file) => file.key === 'config.yaml');
      setConfigYAML(configFile?.content ?? '');
      setDescription(detail.description ?? '');
    }
  }, [isEdit, detail, detailLoaded]);

  // Reset the form when the target changes. New templates start from the
  // starter skeleton; edits start empty and prefill from the detail.
  React.useEffect(() => {
    if (target) {
      prefillApplied.current = false;
      setName(isEdit ? (editTemplate?.name ?? '') : '');
      setDescription('');
      setConfigYAML(isEdit ? '' : STARTER_CONFIG_YAML);
      setError(null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [target]);

  const nameValid = isEdit || TEMPLATE_NAME_PATTERN.test(name);
  const configValid = hasConfigContent(configYAML);
  const isSubmitDisabled =
    !target || !nameValid || !configValid || isSubmitting || (isEdit && !detailLoaded);

  const onSubmit = React.useCallback(async () => {
    if (!target || !nameValid || !configValid) {
      return;
    }
    setIsSubmitting(true);
    setError(null);
    try {
      if (isEdit && editTemplate) {
        const updated = await updateGuardrailTemplate(
          '',
          editTemplate.namespace,
          editTemplate.name,
          {
            configYAML,
            ...(description.trim() ? { description: description.trim() } : {}),
          },
        );
        notification.success(
          'Template updated',
          `Guardrail template ${editTemplate.name} was updated.`,
        );
        onSaved(updated);
      } else {
        const created = await createGuardrailTemplate('', {
          name,
          namespace: defaultNamespace,
          configYAML,
          ...(description.trim() ? { description: description.trim() } : {}),
        });
        notification.success(
          'Template created',
          `Guardrail template ${name} was created in namespace ${defaultNamespace}.`,
        );
        onSaved(created);
      }
    } catch (e) {
      setError(e instanceof Error ? e : new Error(String(e)));
    } finally {
      setIsSubmitting(false);
    }
  }, [
    target,
    isEdit,
    editTemplate,
    name,
    nameValid,
    configValid,
    configYAML,
    description,
    defaultNamespace,
    notification,
    onSaved,
  ]);

  if (!target) {
    return null;
  }

  return (
    <Modal
      isOpen
      onClose={onClose}
      variant="medium"
      aria-label={isEdit ? 'Edit guardrail template' : 'Upload guardrail template'}
      data-testid="manage-template-modal"
    >
      <ModalHeader
        title={isEdit ? `Edit ${editTemplate?.name}` : 'Upload template'}
        description={
          isEdit
            ? 'Edit the guardrail configuration. The config.yaml is updated in place.'
            : 'Create a guardrail template from a config.yaml in your namespace. Upload a file or write the YAML directly.'
        }
      />
      <ModalBody>
        {isEdit && detailError && (
          <Alert variant="danger" title="Failed to load template contents" isInline>
            {detailError.message}
          </Alert>
        )}
        {isEdit && !detailLoaded && !detailError && (
          <Bullseye data-testid="manage-template-loading">
            <Spinner />
          </Bullseye>
        )}
        {(!isEdit || detailLoaded) && (
          <Form>
            <FormGroup label="Name" isRequired={!isEdit} fieldId="guardrail-template-name">
              <TextInput
                isRequired={!isEdit}
                id="guardrail-template-name"
                name="guardrail-template-name"
                value={name}
                isDisabled={isEdit}
                validated={nameValid ? 'default' : 'error'}
                onChange={(_event, value) => setName(value)}
                placeholder="For example, my-injection-guardrail"
                data-testid="guardrail-template-name-input"
              />
              {!nameValid && (
                <FormHelperText>
                  <HelperText>
                    <HelperTextItem variant="error">
                      Name must be lowercase alphanumeric with hyphens, start and end with an
                      alphanumeric character, and be at most 63 characters.
                    </HelperTextItem>
                  </HelperText>
                </FormHelperText>
              )}
            </FormGroup>
            <FormGroup label="Description" fieldId="guardrail-template-description">
              <TextInput
                id="guardrail-template-description"
                name="guardrail-template-description"
                value={description}
                onChange={(_event, value) => setDescription(value)}
                placeholder="For example, Blocks prompt injection attempts"
                data-testid="guardrail-template-description-input"
              />
            </FormGroup>
            <FormGroup label="config.yaml" isRequired fieldId="guardrail-template-config">
              <GuardrailsCodeEditor
                code={configYAML}
                onCodeChange={(value) => setConfigYAML(value)}
                data-testid="guardrail-template-config-editor"
              />
              {!configValid && (
                <FormHelperText>
                  <HelperText>
                    <HelperTextItem variant="error">
                      A non-empty config.yaml is required.
                    </HelperTextItem>
                  </HelperText>
                </FormHelperText>
              )}
            </FormGroup>
          </Form>
        )}
        {error && (
          <Alert
            variant="danger"
            title={isEdit ? 'Failed to update template' : 'Failed to create template'}
            isInline
            className="pf-v6-u-mt-md"
            data-testid="manage-template-error"
          >
            {error.message}
          </Alert>
        )}
      </ModalBody>
      <ModalFooter>
        <Button
          variant="primary"
          onClick={onSubmit}
          isDisabled={isSubmitDisabled}
          isLoading={isSubmitting}
          data-testid="guardrail-template-submit-button"
        >
          {isEdit ? 'Save' : 'Create'}
        </Button>
        <Button variant="link" onClick={onClose} data-testid="guardrail-template-cancel-button">
          Cancel
        </Button>
      </ModalFooter>
    </Modal>
  );
};

export default ManageTemplateModal;
