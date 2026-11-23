import 'package:flutter_test/flutter_test.dart';
import 'package:ai_house_generator/app/app.dart';

void main() {
  testWidgets('Dashboard renders AI House Generator workspace', (tester) async {
    await tester.pumpWidget(const AiHouseGeneratorApp());

    expect(find.text('Build your next home.'), findsOneWidget);
    expect(find.text('Design workspace'), findsOneWidget);
    expect(find.text('Floor Planner'), findsOneWidget);
    expect(find.text('3D Visualization'), findsOneWidget);
    expect(find.text('BOQ & RAB'), findsOneWidget);
  });
}