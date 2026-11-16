import React from 'react';
import {
  Alert,
  Button,
  Form,
  FormGroup,
  FormHelperText,
  HelperText,
  HelperTextItem,
  MenuToggle,
  Modal,
  ModalBody,
  ModalFooter,
  ModalHeader,
  Select,
  SelectOption,
  SelectList,
  TextInput,
} from '@patternfly/react-core';
import { useNotification } from '~/app/hooks/useNotification';
import { createGuardrailDeployment } from '~/app/api/guardrails';
import { getTemplateDisplayName } from '~/app/utilities/guardrailsUtils';
import { GuardrailTemplate, NamespaceKind } from '~/app/types';

const DEPLOYMENT_NAME_PATTERN = /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/;

type DeployGuardrailModalProps = {
  template: GuardrailTemplate | null;
  defaultNamespace: string;
  namespaces: NamespaceKind[];
  onClose: () => void;
  onDeployed: (namespace: string) => void;
};

const DeployGuardrailModal: React.FC<DeployGuardrailModalProps> = ({
  template,
  defaultNamespace,
  namespaces,
  onClose,
  onDeployed,
}) => {
  const notification = useNotification();
  const [name, setName] = React.useState('');
  // Null means "no explicit choice": follow the page's selected namespace,
  // which may default after the modal opens. This avoids resetting the form
  // when the page namespace arrives late.
  const [namespaceOverride, setNamespaceOverride] = React.useState<string | null>(null);
  const [isNamespaceOpen, setIsNamespaceOpen] = React.useState(false);
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [error, setError] = React.useState<Error | null>(null);

  const namespace = namespaceOverride ?? defaultNamespace;

  const nameValid = DEPLOYMENT_NAME_PATTERN.test(name);

  const isSubmitDisabled = !template || !nameValid || !namespace || isSubmitting;

  React.useEffect(() => {
    if (template) {
      setName('');
      setError(null);
      setNamespaceOverride(null);
    }
  }, [template]);

  const namespaceOptions = React.useMemo(
    () =>
      namespaces.map((ns) => (
        <SelectOption key={ns.name} value={ns.name}>
          {ns.name}
        </SelectOption>
      )),
    [namespaces],
  );

  const onNamespaceSelect = React.useCallback(
    (_event: React.MouseEvent | undefined, value: string | number | undefined) => {
      if (typeof value === 'string') {
        setNamespaceOverride(value);
      }
      setIsNamespaceOpen(false);
    },
    [],
  );

  const onSubmit = React.useCallback(async () => {
    if (!template || !nameValid || !namespace) {
      return;
    }
    setIsSubmitting(true);
    setError(null);
    try {
      await createGuardrailDeployment('', {
        name,
        namespace,
        templateName: template.name,
        templateNamespace: template.namespace,
      });
      notification.success(
        'Deployment created',
        `Guardrail deployment ${name} was created in namespace ${namespace}. It may take a moment for the operator to report its status.`,
      );
      onDeployed(namespace);
    } catch (e) {
      setError(e instanceof Error ? e : new Error(String(e)));
    } finally {
      setIsSubmitting(false);
    }
  }, [template, name, nameValid, namespace, notification, onDeployed]);

  if (!template) {
    return null;
  }

  return (
    <Modal
      isOpen
      onClose={onClose}
      variant="small"
      aria-label="Deploy guardrail"
      data-testid="deploy-guardrail-modal"
    >
      <ModalHeader
        title="Deploy guardrail"
        description={`Deploy the ${getTemplateDisplayName(template.name)} template into one of your namespaces.`}
      />
      <ModalBody>
        <Form>
          <FormGroup label="Deployment name" isRequired fieldId="guardrail-deployment-name">
            <TextInput
              isRequired
              id="guardrail-deployment-name"
              name="guardrail-deployment-name"
              value={name}
              validated={name === '' || nameValid ? 'default' : 'error'}
              onChange={(_event, value) => setName(value)}
              placeholder="For example, my-pii-guardrail"
              data-testid="guardrail-deployment-name-input"
            />
            {name !== '' && !nameValid && (
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
          <FormGroup label="Target namespace" isRequired fieldId="guardrail-deployment-namespace">
            <Select
              isOpen={isNamespaceOpen}
              onOpenChange={(open) => setIsNamespaceOpen(open)}
              onSelect={onNamespaceSelect}
              maxMenuHeight="300px"
              toggle={(toggleRef) => (
                <MenuToggle
                  ref={toggleRef}
                  onClick={() => setIsNamespaceOpen(!isNamespaceOpen)}
                  isExpanded={isNamespaceOpen}
                  isFullWidth
                  data-testid="guardrail-deployment-namespace-select"
                >
                  {namespace || 'Select a namespace'}
                </MenuToggle>
              )}
              selected={namespace}
            >
              <SelectList>{namespaceOptions}</SelectList>
            </Select>
          </FormGroup>
        </Form>
        {error && (
          <Alert
            variant="danger"
            title="Failed to create deployment"
            isInline
            className="pf-v6-u-mt-md"
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
          data-testid="guardrail-deployment-submit-button"
        >
          Deploy
        </Button>
        <Button variant="link" onClick={onClose} data-testid="guardrail-deployment-cancel-button">
          Cancel
        </Button>
      </ModalFooter>
    </Modal>
  );
};

export default DeployGuardrailModal;
