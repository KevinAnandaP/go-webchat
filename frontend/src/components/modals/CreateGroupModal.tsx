'use client';

import React, { useState } from 'react';
import { Users, Check } from 'lucide-react';
import { Modal } from '@/components/ui/Modal';
import { Input } from '@/components/ui/Input';
import { Button } from '@/components/ui/Button';
import { Avatar } from '@/components/ui/Avatar';
import { groupsApi } from '@/lib/api';
import { useChatStore } from '@/stores/chatStore';
import { cn } from '@/lib/utils';

interface CreateGroupModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function CreateGroupModal({ isOpen, onClose }: CreateGroupModalProps) {
  const { contacts, fetchConversations } = useChatStore();
  const [groupName, setGroupName] = useState('');
  const [selectedMembers, setSelectedMembers] = useState<Set<string>>(new Set());
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState('');

  const toggleMember = (memberId: string) => {
    const newSelected = new Set(selectedMembers);
    if (newSelected.has(memberId)) {
      newSelected.delete(memberId);
    } else {
      newSelected.add(memberId);
    }
    setSelectedMembers(newSelected);
  };

  const handleCreate = async () => {
    if (!groupName.trim()) {
      setError('Please enter a group name');
      return;
    }

    if (selectedMembers.size === 0) {
      setError('Please select at least one member');
      return;
    }

    setIsCreating(true);
    setError('');

    try {
      await groupsApi.create({
        name: groupName.trim(),
        member_ids: Array.from(selectedMembers),
      });
      fetchConversations();
      handleClose();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Failed to create group';
      setError(message);
    } finally {
      setIsCreating(false);
    }
  };

  const handleClose = () => {
    setGroupName('');
    setSelectedMembers(new Set());
    setError('');
    onClose();
  };

  return (
    <Modal isOpen={isOpen} onClose={handleClose} title="Create Group">
      <div className="space-y-4">
        <Input
          type="text"
          label="Group Name"
          placeholder="My Awesome Group"
          value={groupName}
          onChange={(e) => setGroupName(e.target.value)}
          leftIcon={<Users className="w-4 h-4" />}
        />

        <div>
          <label className="block text-sm font-medium text-[var(--text-secondary)] mb-2">
            Select Members ({selectedMembers.size} selected)
          </label>
          
          {contacts.length === 0 ? (
            <p className="text-sm text-[var(--text-tertiary)] text-center py-4">
              No contacts available. Add contacts first.
            </p>
          ) : (
            <div className="max-h-48 overflow-y-auto space-y-1 border border-[var(--separator)] rounded-xl p-2">
              {contacts.map((contact) => (
                <button
                  key={contact.id}
                  onClick={() => toggleMember(contact.id)}
                  className={cn(
                    'w-full flex items-center gap-3 p-2 rounded-xl transition-colors',
                    selectedMembers.has(contact.id)
                      ? 'bg-[var(--accent-blue)]/10'
                      : 'hover:bg-[var(--bg-secondary)]'
                  )}
                >
                  <Avatar src={contact.avatar} name={contact.name} size="sm" />
                  <span className="flex-1 text-left text-sm text-[var(--text-primary)]">
                    {contact.name}
                  </span>
                  {selectedMembers.has(contact.id) && (
                    <Check className="w-4 h-4 text-[var(--accent-blue)]" />
                  )}
                </button>
              ))}
            </div>
          )}
        </div>

        {error && (
          <p className="text-sm text-[var(--accent-red)]">{error}</p>
        )}

        <div className="flex gap-2 pt-2">
          <Button variant="secondary" onClick={handleClose} className="flex-1">
            Cancel
          </Button>
          <Button
            onClick={handleCreate}
            isLoading={isCreating}
            disabled={!groupName.trim() || selectedMembers.size === 0}
            className="flex-1"
          >
            Create Group
          </Button>
        </div>
      </div>
    </Modal>
  );
}
