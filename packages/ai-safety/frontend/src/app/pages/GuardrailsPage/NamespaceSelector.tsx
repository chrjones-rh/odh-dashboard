import React from 'react';
import {
  Flex,
  FlexItem,
  MenuToggle,
  Select,
  SelectOption,
  SelectList,
  Spinner,
} from '@patternfly/react-core';
import { FolderOpenIcon } from '@patternfly/react-icons';
import { NamespaceKind } from '~/app/types';

type NamespaceSelectorProps = {
  namespaces: NamespaceKind[];
  namespace: string;
  onChange: (namespace: string) => void;
  loaded: boolean;
};

const NamespaceSelector: React.FC<NamespaceSelectorProps> = ({
  namespaces,
  namespace,
  onChange,
  loaded,
}) => {
  const [isOpen, setIsOpen] = React.useState(false);

  const options = React.useMemo(
    () =>
      namespaces.map((ns) => (
        <SelectOption key={ns.name} value={ns.name}>
          {ns.name}
        </SelectOption>
      )),
    [namespaces],
  );

  const onSelect = React.useCallback(
    (_event: React.MouseEvent | undefined, value: string | number | undefined) => {
      if (typeof value === 'string') {
        onChange(value);
      }
      setIsOpen(false);
    },
    [onChange],
  );

  return (
    <Flex
      alignItems={{ default: 'alignItemsCenter' }}
      flexWrap={{ default: 'nowrap' }}
      gap={{ default: 'gapSm' }}
    >
      <FlexItem>
        <FolderOpenIcon />
      </FlexItem>
      <FlexItem>Project</FlexItem>
      <FlexItem>
        {loaded ? (
          <Select
            isOpen={isOpen}
            onOpenChange={(open) => setIsOpen(open)}
            onSelect={onSelect}
            maxMenuHeight="300px"
            toggle={(toggleRef) => (
              <MenuToggle
                ref={toggleRef}
                onClick={() => setIsOpen(!isOpen)}
                isExpanded={isOpen}
                data-testid="guardrails-namespace-select"
              >
                {namespace || 'Select a project'}
              </MenuToggle>
            )}
            selected={namespace}
          >
            <SelectList>{options}</SelectList>
          </Select>
        ) : (
          <Spinner size="sm" data-testid="guardrails-namespace-select-loading" />
        )}
      </FlexItem>
    </Flex>
  );
};

export default NamespaceSelector;
