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
  Spinner,
  TextInput,
} from '@patternfly/react-core';
import { useNotification } from '~/app/hooks/useNotification';
import { createGuardrailDeployment } from '~/app/api/guardrails';
import { useServingModels } from '~/app/hooks/useGuardrails';
import { getTemplateDisplayName } from '~/app/utilities/guardrailsUtils';
import { GuardrailTemplate, NamespaceKind } from '~/app/types';

const DEPLOYMENT_NAME_PATTERN = /^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/;
const CUSTOM_MODEL_KEY = '__custom__';
const NO_MODEL_KEY = '__none__';

const isValidEndpoint = (value: string): boolean => {
  try {
    const url = new URL(value);
    return (url.protocol === 'http:' || url.protocol === 'https:') && url.hostname !== '';
  } catch {
    return false;
  }
};

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
  const [selectedModelKey, setSelectedModelKey] = React.useState<string>(NO_MODEL_KEY);
  const [isModelOpen, setIsModelOpen] = React.useState(false);
  const [modelName, setModelName] = React.useState('');
  const [modelEndpoint, setModelEndpoint] = React.useState('');
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [error, setError] = React.useState<Error | null>(null);

  const namespace = namespaceOverride ?? defaultNamespace;
  const [servingModels, modelsLoaded, modelsLoadError] = useServingModels(namespace);

  const nameValid = DEPLOYMENT_NAME_PATTERN.test(name);

  const modelWiringStarted = modelName.trim() !== '' || modelEndpoint.trim() !== '';
  const modelValid =
    !modelWiringStarted || (modelName.trim() !== '' && isValidEndpoint(modelEndpoint.trim()));

  const isSubmitDisabled = !template || !nameValid || !namespace || !modelValid || isSubmitting;

  React.useEffect(() => {
    if (template) {
      setName('');
      setError(null);
      setNamespaceOverride(null);
      setSelectedModelKey(NO_MODEL_KEY);
      setModelName('');
      setModelEndpoint('');
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

  const onModelSelect = React.useCallback(
    (_event: React.MouseEvent | undefined, value: string | number | undefined) => {
      if (typeof value !== 'string') {
        return;
      }
      setSelectedModelKey(value);
      setIsModelOpen(false);
      if (value === NO_MODEL_KEY) {
        setModelName('');
        setModelEndpoint('');
      } else if (value === CUSTOM_MODEL_KEY) {
        setModelName('');
        setModelEndpoint('');
      } else {
        const model = servingModels.find((m) => `${m.kind}/${m.resourceName}` === value);
        if (model) {
          setModelName(model.name);
          setModelEndpoint(model.endpoint ?? '');
        }
      }
    },
    [servingModels],
  );

  const modelOptions = React.useMemo(() => {
    const options: React.ReactNode[] = [
      <SelectOption key={NO_MODEL_KEY} value={NO_MODEL_KEY}>
        None (configure later)
      </SelectOption>,
    ];
    if (modelsLoaded) {
      if (servingModels.length === 0) {
        options.push(
          <SelectOption key="no-models" isDisabled>
            No models found in {namespace || 'this namespace'}
          </SelectOption>,
        );
      }
      servingModels.forEach((model) => {
        const key = `${model.kind}/${model.resourceName}`;
        options.push(
          <SelectOption
            key={key}
            value={key}
            isDisabled={!model.ready || !model.endpoint}
            description={`${model.kind}${model.ready ? '' : ' · not ready'}`}
          >
            {model.name}
          </SelectOption>,
        );
      });
    }
    options.push(
      <SelectOption key={CUSTOM_MODEL_KEY} value={CUSTOM_MODEL_KEY}>
        Custom model…
      </SelectOption>,
    );
    return options;
  }, [servingModels, modelsLoaded, namespace]);

  const selectedModelLabel = React.useMemo(() => {
    if (selectedModelKey === NO_MODEL_KEY) {
      return 'None (configure later)';
    }
    if (selectedModelKey === CUSTOM_MODEL_KEY) {
      return 'Custom model…';
    }
    const model = servingModels.find((m) => `${m.kind}/${m.resourceName}` === selectedModelKey);
    return model ? model.name : selectedModelKey;
  }, [selectedModelKey, servingModels]);

  const onSubmit = React.useCallback(async () => {
    if (!template || !nameValid || !namespace || !modelValid) {
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
        ...(modelWiringStarted
          ? { model: { name: modelName.trim(), endpoint: modelEndpoint.trim() } }
          : {}),
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
  }, [
    template,
    name,
    nameValid,
    namespace,
    modelValid,
    modelWiringStarted,
    modelName,
    modelEndpoint,
    notification,
    onDeployed,
  ]);

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
              popperProps={{ appendTo: 'inline' }}
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
          <FormGroup label="Model" fieldId="guardrail-deployment-model">
            <Select
              isOpen={isModelOpen}
              onOpenChange={(open) => setIsModelOpen(open)}
              onSelect={onModelSelect}
              maxMenuHeight="300px"
              popperProps={{ appendTo: 'inline' }}
              toggle={(toggleRef) => (
                <MenuToggle
                  ref={toggleRef}
                  onClick={() => setIsModelOpen(!isModelOpen)}
                  isExpanded={isModelOpen}
                  isFullWidth
                  data-testid="guardrail-deployment-model-select"
                >
                  {modelsLoaded ? (
                    selectedModelLabel
                  ) : (
                    <Spinner size="sm" aria-label="Loading models" />
                  )}
                </MenuToggle>
              )}
              selected={selectedModelKey}
            >
              <SelectList>{modelOptions}</SelectList>
            </Select>
            <FormHelperText>
              <HelperText>
                <HelperTextItem>
                  The model the guardrail passes allowed messages to. Wiring a model lets you test
                  the deployment from the deployments list.
                </HelperTextItem>
              </HelperText>
            </FormHelperText>
            {modelsLoadError && (
              <FormHelperText>
                <HelperText>
                  <HelperTextItem variant="warning">
                    Models in {namespace || 'the namespace'} could not be listed. You can still
                    enter a custom model.
                  </HelperTextItem>
                </HelperText>
              </FormHelperText>
            )}
          </FormGroup>
          {selectedModelKey !== NO_MODEL_KEY && (
            <>
              <FormGroup
                label="Model name"
                isRequired={modelWiringStarted}
                fieldId="guardrail-deployment-model-name"
              >
                <TextInput
                  id="guardrail-deployment-model-name"
                  name="guardrail-deployment-model-name"
                  value={modelName}
                  onChange={(_event, value) => setModelName(value)}
                  placeholder="For example, Qwen/Qwen3-VL-30B-A3B-Instruct"
                  data-testid="guardrail-deployment-model-name-input"
                />
              </FormGroup>
              <FormGroup
                label="Model endpoint"
                isRequired={modelWiringStarted}
                fieldId="guardrail-deployment-model-endpoint"
              >
                <TextInput
                  id="guardrail-deployment-model-endpoint"
                  name="guardrail-deployment-model-endpoint"
                  value={modelEndpoint}
                  validated={
                    modelEndpoint === '' || isValidEndpoint(modelEndpoint.trim())
                      ? 'default'
                      : 'error'
                  }
                  onChange={(_event, value) => setModelEndpoint(value)}
                  placeholder="For example, https://my-model-predictor.my-project.svc.cluster.local:8080/v1"
                  data-testid="guardrail-deployment-model-endpoint-input"
                />
                {modelEndpoint !== '' && !isValidEndpoint(modelEndpoint.trim()) && (
                  <FormHelperText>
                    <HelperText>
                      <HelperTextItem variant="error">
                        Endpoint must be an http or https URL, including the /v1 suffix.
                      </HelperTextItem>
                    </HelperText>
                  </FormHelperText>
                )}
              </FormGroup>
            </>
          )}
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
