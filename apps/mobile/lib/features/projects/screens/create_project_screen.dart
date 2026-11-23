import 'package:flutter/material.dart';

import '../../../core/theme/app_theme.dart';
import '../services/projects_api.dart';

class CreateProjectScreen extends StatefulWidget {
  const CreateProjectScreen({super.key, required this.api});

  final ProjectsApi api;

  @override
  State<CreateProjectScreen> createState() => _CreateProjectScreenState();
}

class _CreateProjectScreenState extends State<CreateProjectScreen> {
  final _formKey = GlobalKey<FormState>();
  final _name = TextEditingController();
  final _description = TextEditingController();
  final _location = TextEditingController();
  final _width = TextEditingController();
  final _depth = TextEditingController();

  String _orientation = 'N';
  int _floors = 1;
  bool _saving = false;

  @override
  void dispose() {
    _name.dispose();
    _description.dispose();
    _location.dispose();
    _width.dispose();
    _depth.dispose();
    super.dispose();
  }

  String? _requiredText(String? value, String label, int maxLength) {
    final text = value?.trim() ?? '';

    if (text.isEmpty) return '$label is required.';
    if (text.length > maxLength) return '$label is too long.';

    return null;
  }

  String? _dimensionValidator(String? value, String label) {
    final parsed = double.tryParse((value ?? '').trim());

    if (parsed == null || !parsed.isFinite || parsed < 1 || parsed > 1000) {
      return '$label must be between 1 and 1000 metres.';
    }

    return null;
  }

  Future<void> _submit() async {
    if (_saving || !_formKey.currentState!.validate()) return;

    setState(() => _saving = true);

    try {
      final id = await widget.api.createProject(
        name: _name.text.trim(),
        description: _description.text.trim(),
        location: _location.text.trim(),
        landWidth: double.parse(_width.text.trim()),
        landDepth: double.parse(_depth.text.trim()),
        orientation: _orientation,
        floors: _floors,
      );

      if (!mounted) return;
      Navigator.of(context).pop(id);
    } on ProjectsApiException catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(error.message)));
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  InputDecoration _decoration(String label, {String? hint}) {
    return InputDecoration(
      labelText: label,
      hintText: hint,
      filled: true,
      fillColor: AppColors.surface,
      border: OutlineInputBorder(
        borderRadius: BorderRadius.circular(11),
        borderSide: const BorderSide(color: AppColors.border),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(11),
        borderSide: const BorderSide(color: AppColors.border),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: AppBar(title: const Text('Create project')),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.all(20),
          children: [
            Text(
              'Tell us about your land.',
              style: Theme.of(context).textTheme.headlineSmall,
            ),
            const SizedBox(height: 7),
            const Text(
              'Start with the basic project and site information. '
              'You can refine the design later.',
              style: TextStyle(color: AppColors.muted, height: 1.5),
            ),
            const SizedBox(height: 22),
            TextFormField(
              controller: _name,
              maxLength: 200,
              decoration: _decoration('Project name'),
              validator: (value) => _requiredText(value, 'Project name', 200),
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _description,
              maxLength: 5000,
              maxLines: 3,
              decoration: _decoration('Description', hint: 'Optional details'),
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _location,
              maxLength: 300,
              decoration: _decoration('Location'),
              validator: (value) => _requiredText(value, 'Location', 300),
            ),
            const SizedBox(height: 22),
            Text(
              'Land dimensions',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: TextFormField(
                    controller: _width,
                    keyboardType: const TextInputType.numberWithOptions(
                      decimal: true,
                    ),
                    decoration: _decoration('Width (m)'),
                    validator: (value) => _dimensionValidator(value, 'Width'),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: TextFormField(
                    controller: _depth,
                    keyboardType: const TextInputType.numberWithOptions(
                      decimal: true,
                    ),
                    decoration: _decoration('Depth (m)'),
                    validator: (value) => _dimensionValidator(value, 'Depth'),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<String>(
              initialValue: _orientation,
              decoration: _decoration('Land orientation'),
              items: const [
                DropdownMenuItem(value: 'N', child: Text('North')),
                DropdownMenuItem(value: 'NE', child: Text('North East')),
                DropdownMenuItem(value: 'E', child: Text('East')),
                DropdownMenuItem(value: 'SE', child: Text('South East')),
                DropdownMenuItem(value: 'S', child: Text('South')),
                DropdownMenuItem(value: 'SW', child: Text('South West')),
                DropdownMenuItem(value: 'W', child: Text('West')),
                DropdownMenuItem(value: 'NW', child: Text('North West')),
              ],
              onChanged: (value) {
                if (value != null) setState(() => _orientation = value);
              },
            ),
            const SizedBox(height: 12),
            DropdownButtonFormField<int>(
              initialValue: _floors,
              decoration: _decoration('Number of floors'),
              items: const [
                DropdownMenuItem(value: 1, child: Text('1 floor')),
                DropdownMenuItem(value: 2, child: Text('2 floors')),
              ],
              onChanged: (value) {
                if (value != null) setState(() => _floors = value);
              },
            ),
            const SizedBox(height: 24),
            SizedBox(
              height: 50,
              child: FilledButton.icon(
                onPressed: _saving ? null : _submit,
                icon: _saving
                    ? const SizedBox(
                        width: 18,
                        height: 18,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const Icon(Icons.add_home_work_outlined),
                label: Text(_saving ? 'Creating project...' : 'Create project'),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
